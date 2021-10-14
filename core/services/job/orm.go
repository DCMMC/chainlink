package job

import (
	"context"
	"fmt"
	"time"

	"github.com/smartcontractkit/chainlink/core/chains/evm"
	"github.com/smartcontractkit/chainlink/core/logger"
	"github.com/smartcontractkit/chainlink/core/services/keystore"
	"github.com/smartcontractkit/chainlink/core/services/pipeline"
	"github.com/smartcontractkit/chainlink/core/services/postgres"
	"github.com/smartcontractkit/chainlink/core/store/models"
	"github.com/smartcontractkit/sqlx"
	"go.uber.org/multierr"
	"gorm.io/gorm"

	"github.com/jackc/pgconn"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

var (
	ErrNoSuchPeerID             = errors.New("no such peer id exists")
	ErrNoSuchKeyBundle          = errors.New("no such key bundle exists")
	ErrNoSuchTransmitterAddress = errors.New("no such transmitter address exists")
	ErrNoSuchPublicKey          = errors.New("no such public key exists")
)

//go:generate mockery --name ORM --output ./mocks/ --case=underscore

type ORM interface {
	CreateJob(ctx context.Context, jobSpec *Job, pipeline pipeline.Pipeline) (Job, error)
	FindJobs(offset, limit int) ([]Job, int, error)
	FindJobTx(id int32) (Job, error)
	FindJob(ctx context.Context, id int32) (Job, error)
	FindJobByExternalJobID(ctx context.Context, uuid uuid.UUID) (Job, error)
	FindJobIDsWithBridge(name string) ([]int32, error)
	DeleteJob(ctx context.Context, id int32) error
	RecordError(ctx context.Context, jobID int32, description string)
	DismissError(ctx context.Context, errorID int32) error
	Close() error
	PipelineRuns(jobID *int32, offset, size int) ([]pipeline.Run, int, error)
}

type orm struct {
	db          *sqlx.DB
	chainSet    evm.ChainSet
	keyStore    keystore.Master
	pipelineORM pipeline.ORM
	lggr        logger.Logger
}

var _ ORM = (*orm)(nil)

func NewORM(
	db *sqlx.DB,
	chainSet evm.ChainSet,
	pipelineORM pipeline.ORM,
	keyStore keystore.Master, // needed to validation key properties on new job creation
	lggr logger.Logger,
) *orm {
	return &orm{
		db:          db,
		chainSet:    chainSet,
		keyStore:    keyStore,
		pipelineORM: pipelineORM,
		lggr:        lggr.Named("ORM"),
	}
}

// NOTE: N+1 query
func LoadAllJobsTypes(tx *sqlx.Tx, jobs []Job) error {
	for i := range jobs {
		err := LoadAllJobTypes(tx, &jobs[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func LoadAllJobTypes(tx *sqlx.Tx, job *Job) error {
	return multierr.Combine(
		loadJobType(tx, job.PipelineSpec, "pipeline_specs", &job.PipelineSpecID),
		loadJobType(tx, job.FluxMonitorSpec, "flux_monitor_specs", job.FluxMonitorSpecID),
		loadJobType(tx, job.DirectRequestSpec, "direct_request_specs", job.DirectRequestSpecID),
		loadJobType(tx, job.OffchainreportingOracleSpec, "offchainreporting_oracle_specs", job.OffchainreportingOracleSpecID),
		loadJobType(tx, job.KeeperSpec, "keeper_specs", job.KeeperSpecID),
		loadJobType(tx, job.CronSpec, "cron_specs", job.CronSpecID),
		loadJobType(tx, job.WebhookSpec, "webhook_specs", job.WebhookSpecID),
		loadJobType(tx, job.VRFSpec, "vrf_specs", job.VRFSpecID),
	)
}

func loadJobType(tx *sqlx.Tx, dest interface{}, table string, id *int32) error {
	if id == nil {
		return nil
	}
	err := tx.Get(dest, fmt.Sprintf(`SELECT * FROM %s WHERE id = $1`, table), *id)
	return errors.Wrapf(err, "failed to load job type %s with id %d", table, *id)
}

func (o *orm) Close() error {
	return nil
}

// CreateJob creates the job and it's associated spec record.
//
// NOTE: This is not wrapped in a db transaction so if you call this, you should
// use postgres.TransactionManager to create the transaction in the context.
// Expects an unmarshaled job spec as the jobSpec argument i.e. output from ValidatedXX.
// Returns a fully populated Job.
func (o *orm) CreateJob(ctx context.Context, jobSpec *Job, p pipeline.Pipeline) (Job, error) {
	postgres.EnsureNoTxInContext(ctx)
	var jb Job
	for _, task := range p.Tasks {
		if task.Type() == pipeline.TaskTypeBridge {
			// Bridge must exist
			name := task.(*pipeline.BridgeTask).Name

			sql := `SELECT EXISTS(SELECT 1 FROM bridge_types WHERE name = $1);`
			var exists bool
			err := o.db.QueryRowx(sql, name).Scan(&exists)
			if err != nil {
				return jb, err
			}
			if !exists {
				return jb, errors.Wrap(pipeline.ErrNoSuchBridge, name)
			}
		}
	}

	err := postgres.SqlxTransaction(ctx, o.db, func(tx *sqlx.Tx) error {

		// Autogenerate a job ID if not specified
		if jobSpec.ExternalJobID == (uuid.UUID{}) {
			jobSpec.ExternalJobID = uuid.NewV4()
		}

		switch jobSpec.Type {
		case DirectRequest:
			sql := `INSERT INTO direct_request_specs (contract_address, min_incoming_confirmations, requesters, min_contract_payment, evm_chain_id, created_at, updated_at)
			VALUES (:contract_address, :min_incoming_confirmations, :requesters, :min_contract_payment, :evm_chain_id, now(), now())
			RETURNING *;`
			err := postgres.PrepareGet(tx, sql, &jobSpec.DirectRequestSpec, &jobSpec.DirectRequestSpec)
			if err != nil {
				return errors.Wrap(err, "failed to create DirectRequestSpec for jobSpec")
			}
			jobSpec.DirectRequestSpecID = &jobSpec.DirectRequestSpec.ID
		case FluxMonitor:
			sql := `INSERT INTO flux_monitor_specs (contract_address, threshold, absolute_threshold, poll_timer_period, poll_timer_disabled, idle_timer_period, idle_timer_disabled 
					drumbeat_schedule, drumbeat_random_delay, drumbeat_enabled, min_payment, evm_chain_id, created_at, updated_at)
			VALUES (:contract_address, :threshold, :absolute_threshold, :poll_timer_period, :poll_timer_disabled, :idle_timer_period, :idle_timer_disabled 
					:drumbeat_schedule, :drumbeat_random_delay, :drumbeat_enabled, :min_payment, :evm_chain_id, NOW(), NOW())
			RETURNING *;`
			err := postgres.PrepareGet(tx, sql, &jobSpec.FluxMonitorSpec, &jobSpec.FluxMonitorSpec)
			if err != nil {
				return errors.Wrap(err, "failed to create FluxMonitorSpec for jobSpec")
			}
			jobSpec.FluxMonitorSpecID = &jobSpec.FluxMonitorSpec.ID
		case OffchainReporting:
			if jobSpec.OffchainreportingOracleSpec.EncryptedOCRKeyBundleID != nil {
				_, err := o.keyStore.OCR().Get(jobSpec.OffchainreportingOracleSpec.EncryptedOCRKeyBundleID.String())
				if err != nil {
					return errors.Wrapf(ErrNoSuchKeyBundle, "%v", jobSpec.OffchainreportingOracleSpec.EncryptedOCRKeyBundleID)
				}
			}
			if jobSpec.OffchainreportingOracleSpec.P2PPeerID != nil {
				_, err := o.keyStore.P2P().Get(jobSpec.OffchainreportingOracleSpec.P2PPeerID.Raw())
				if err != nil {
					return errors.Wrapf(ErrNoSuchPeerID, "%v", jobSpec.OffchainreportingOracleSpec.P2PPeerID)
				}
			}
			if jobSpec.OffchainreportingOracleSpec.TransmitterAddress != nil {
				_, err := o.keyStore.Eth().Get(jobSpec.OffchainreportingOracleSpec.TransmitterAddress.Hex())
				if err != nil {
					return errors.Wrapf(ErrNoSuchTransmitterAddress, "%v", jobSpec.OffchainreportingOracleSpec.TransmitterAddress)
				}
			}

			sql := `INSERT INTO offchainreporting_oracle_specs (contract_address, p2p_peer_id, p2p_bootstrap_peers, is_bootstrap_peer, encrypted_ocr_key_bundle_id, transmitter_address,
					observation_timeout, blockchain_timeout, contract_config_tracker_subscribe_interval, contract_config_tracker_poll_interval, contract_config_confirmations, evm_chain_id,
					created_at, updated_at)
			VALUES (:contract_address, :p2p_peer_id, :p2p_bootstrap_peers, :is_bootstrap_peer, :encrypted_ocr_key_bundle_id, :transmitter_address,
					:observation_timeout, :blockchain_timeout, :contract_config_tracker_subscribe_interval, :contract_config_tracker_poll_interval, :contract_config_confirmations, evm_chain_id,
					NOW(), NOW())
			RETURNING *;`
			err := postgres.PrepareGet(tx, sql, &jobSpec.OffchainreportingOracleSpec, &jobSpec.OffchainreportingOracleSpec)
			if err != nil {
				return errors.Wrap(err, "failed to create OffchainreportingOracleSpec for jobSpec")
			}
			jobSpec.OffchainreportingOracleSpecID = &jobSpec.OffchainreportingOracleSpec.ID
		case Keeper:
			sql := `INSERT INTO keeper_specs (contract_address, from_address, evm_chain_id, created_at, updated_at)
			VALUES (:contract_address, :from_address, :evm_chain_id, NOW(), NOW())
			RETURNING *;`
			err := postgres.PrepareGet(tx, sql, &jobSpec.KeeperSpec, &jobSpec.KeeperSpec)
			if err != nil {
				return errors.Wrap(err, "failed to create KeeperSpec for jobSpec")
			}
			jobSpec.KeeperSpecID = &jobSpec.KeeperSpec.ID
		case Cron:
			sql := `INSERT INTO cron_specs (cron_schedule, created_at, updated_at)
			VALUES (:cron_schedule, NOW(), NOW())
			RETURNING *;`
			err := postgres.PrepareGet(tx, sql, &jobSpec.CronSpec, &jobSpec.CronSpec)
			if err != nil {
				return errors.Wrap(err, "failed to create CronSpec for jobSpec")
			}
			jobSpec.CronSpecID = &jobSpec.CronSpec.ID
		case VRF:
			sql := `INSERT INTO vrf_specs (coordinator_address, public_key, confirmations, evm_chain_id, created_at, updated_at)
			VALUES (:coordinator_address, :public_key, :confirmations, :evm_chain_id, NOW(), NOW())
			RETURNING *;`
			err := postgres.PrepareGet(tx, sql, &jobSpec.VRFSpec, &jobSpec.VRFSpec)
			pqErr, ok := err.(*pgconn.PgError)
			if err != nil && ok && pqErr.Code == "23503" {
				if pqErr.ConstraintName == "vrf_specs_public_key_fkey" {
					return errors.Wrapf(ErrNoSuchPublicKey, "%s", jobSpec.VRFSpec.PublicKey.String())
				}
			}
			if err != nil {
				return errors.Wrap(err, "failed to create VRFSpec for jobSpec")
			}
			jobSpec.VRFSpecID = &jobSpec.VRFSpec.ID
		case Webhook:
			sql := `INSERT INTO webhook_specs (created_at, updated_at)
			VALUES (NOW(), NOW())
			RETURNING *;`
			err := postgres.PrepareGet(tx, sql, &jobSpec.WebhookSpec, &jobSpec.WebhookSpec)
			if err != nil {
				return errors.Wrap(err, "failed to create WebhookSpec for jobSpec")
			}
			jobSpec.WebhookSpecID = &jobSpec.WebhookSpec.ID

			for i := range jobSpec.WebhookSpec.ExternalInitiatorWebhookSpecs {
				jobSpec.WebhookSpec.ExternalInitiatorWebhookSpecs[i].WebhookSpecID = jobSpec.WebhookSpec.ID
			}
			sqlEI := `INSERT INTO external_initiator_webhook_specs (external_initiator_id, webhook_spec_id, spec)
			VALUES (:external_initiator_id, :webhook_spec_id, :spec)
			RETURNING *;`
			err = postgres.PrepareGet(tx, sqlEI, &jobSpec.WebhookSpec.ExternalInitiatorWebhookSpecs, &jobSpec.WebhookSpec.ExternalInitiatorWebhookSpecs)
			if err != nil {
				return errors.Wrap(err, "failed to create WebhookSpec for jobSpec")
			}
		default:
			logger.Fatalf("Unsupported jobSpec.Type: %v", jobSpec.Type)
		}

		pipelineSpecID, err := o.pipelineORM.CreateSpec(tx, p, jobSpec.MaxTaskDuration)
		if err != nil {
			return errors.Wrap(err, "failed to create pipeline spec")
		}
		jobSpec.PipelineSpecID = pipelineSpecID

		sql := `INSERT INTO jobs (pipeline_spec_id, offchainreporting_oracle_spec_id, name, schema_version, type, max_task_duration, direct_request_spec_id, flux_monitor_spec_id,
				keeper_spec_id, cron_spec_id, vrf_spec_id, webhook_spec_id, external_job_id, created_at, updated_at)
		VALUES (:pipeline_spec_id, :offchainreporting_oracle_spec_id, :name, :schema_version, :type, :max_task_duration, :direct_request_spec_id, :flux_monitor_spec_id,
				:keeper_spec_id, :cron_spec_id, :vrf_spec_id, :webhook_spec_id, :external_job_id, NOW(), NOW())
		RETURNING *;`
		err = postgres.PrepareGet(tx, sql, &jobSpec, &jobSpec)
		return errors.Wrap(err, "failed to create job")
	})
	if err != nil {
		return jb, err
	}

	return o.FindJob(ctx, jobSpec.ID)
}

// DeleteJob removes a job
func (o *orm) DeleteJob(ctx context.Context, id int32) error {
	postgres.EnsureNoTxInContext(ctx)
	sql := `
		WITH deleted_jobs AS (
			DELETE FROM jobs WHERE id = ? RETURNING
				pipeline_spec_id,
				offchainreporting_oracle_spec_id,
				keeper_spec_id,
				cron_spec_id,
				flux_monitor_spec_id,
				vrf_spec_id,
				webhook_spec_id,
				direct_request_spec_id
		),
		deleted_oracle_specs AS (
			DELETE FROM offchainreporting_oracle_specs WHERE id IN (SELECT offchainreporting_oracle_spec_id FROM deleted_jobs)
		),
		deleted_keeper_specs AS (
			DELETE FROM keeper_specs WHERE id IN (SELECT keeper_spec_id FROM deleted_jobs)
		),
		deleted_cron_specs AS (
			DELETE FROM cron_specs WHERE id IN (SELECT cron_spec_id FROM deleted_jobs)
		),
		deleted_fm_specs AS (
			DELETE FROM flux_monitor_specs WHERE id IN (SELECT flux_monitor_spec_id FROM deleted_jobs)
		),
		deleted_vrf_specs AS (
			DELETE FROM vrf_specs WHERE id IN (SELECT vrf_spec_id FROM deleted_jobs)
		),
		deleted_webhook_specs AS (
			DELETE FROM webhook_specs WHERE id IN (SELECT webhook_spec_id FROM deleted_jobs)
		),
		deleted_dr_specs AS (
			DELETE FROM direct_request_specs WHERE id IN (SELECT direct_request_spec_id FROM deleted_jobs)
		)
		DELETE FROM pipeline_specs WHERE id IN (SELECT pipeline_spec_id FROM deleted_jobs)`
	_, err := o.db.ExecContext(ctx, sql, id)
	if err != nil {
		return errors.Wrap(err, "DeleteJob failed to delete job")
	}
	return nil
}

func (o *orm) RecordError(ctx context.Context, jobID int32, description string) {
	postgres.EnsureNoTxInContext(ctx)
	sql := `INSERT INTO job_spec_errors (job_id, description, occurrences, created_at, updated_at)
	VALUES ($1, $2, 1, NOW(), NOW())
	ON CONFLICT (job_id, description) DO UPDATE SET
	occurrences = job_spec_errors.occurrences + 1
	updated_at = excluded.updated_at`
	_, err := o.db.ExecContext(ctx, sql, jobID, description)
	// Noop if the job has been deleted.
	pqErr, ok := err.(*pgconn.PgError)
	if err != nil && ok && pqErr.Code == "23503" {
		if pqErr.ConstraintName == "job_spec_errors_v2_job_id_fkey" {
			return
		}
	}
	o.lggr.ErrorIf(err, fmt.Sprintf("Error creating SpecError %v", description))
}

func (o *orm) DismissError(ctx context.Context, ID int32) error {
	postgres.EnsureNoTxInContext(ctx)
	res, err := o.db.ExecContext(ctx, "DELETE FROM job_spec_errors WHERE id = $1", ID)
	if err != nil {
		return errors.Wrap(err, "failed to dismiss error")
	}
	n, err := res.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to dismiss error")
	}
	if n == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (o *orm) FindJobs(offset, limit int) (jobs []Job, count int, err error) {
	err = postgres.SqlxTransactionWithDefaultCtx(o.db, func(tx *sqlx.Tx) error {
		sql := `SELECT count(*) FROM jobs;`
		err = tx.QueryRowx(sql).Scan(&count)
		if err != nil {
			return err
		}

		sql = `SELECT * FROM jobs ORDER BY id ASC OFFSET $1 LIMIT $2;`
		err = tx.Select(&jobs, sql, offset, limit)
		if err != nil {
			return err
		}

		err = LoadAllJobsTypes(tx, jobs)
		if err != nil {
			return err
		}
		for i := range jobs {
			if jobs[i].OffchainreportingOracleSpec != nil {
				ch, err := o.chainSet.Get(jobs[i].OffchainreportingOracleSpec.EVMChainID.ToInt())
				if err != nil {
					return err
				}
				jobs[i].OffchainreportingOracleSpec = LoadDynamicConfigVars(ch.Config(), *jobs[i].OffchainreportingOracleSpec)
			}
		}
		return nil
	})
	return jobs, int(count), err
}

type OCRSpecConfig interface {
	OCRBlockchainTimeout() time.Duration
	OCRContractConfirmations() uint16
	OCRContractPollInterval() time.Duration
	OCRContractSubscribeInterval() time.Duration
	OCRObservationTimeout() time.Duration
}

func LoadDynamicConfigVars(cfg OCRSpecConfig, os OffchainReportingOracleSpec) *OffchainReportingOracleSpec {
	if os.ObservationTimeout == 0 {
		os.ObservationTimeout = models.Interval(cfg.OCRObservationTimeout())
	}
	if os.BlockchainTimeout == 0 {
		os.BlockchainTimeout = models.Interval(cfg.OCRBlockchainTimeout())
	}
	if os.ContractConfigTrackerSubscribeInterval == 0 {
		os.ContractConfigTrackerSubscribeInterval = models.Interval(cfg.OCRContractSubscribeInterval())
	}
	if os.ContractConfigTrackerPollInterval == 0 {
		os.ContractConfigTrackerPollInterval = models.Interval(cfg.OCRContractPollInterval())
	}
	if os.ContractConfigConfirmations == 0 {
		os.ContractConfigConfirmations = cfg.OCRContractConfirmations()
	}
	return &os
}

func (o *orm) FindJobTx(id int32) (Job, error) {
	ctx, cancel := postgres.DefaultQueryCtx()
	defer cancel()
	return o.FindJob(ctx, id)
}

// FindJob returns job by ID
func (o *orm) FindJob(ctx context.Context, id int32) (jb Job, err error) {
	return o.findJob(ctx, "id", id)
}

func (o *orm) FindJobByExternalJobID(ctx context.Context, externalJobID uuid.UUID) (jb Job, err error) {
	return o.findJob(ctx, "external_job_id", externalJobID)
}

func (o *orm) findJob(ctx context.Context, col string, arg interface{}) (jb Job, err error) {
	postgres.EnsureNoTxInContext(ctx)

	err = postgres.SqlxTransaction(ctx, o.db, func(tx *sqlx.Tx) error {
		sql := fmt.Sprintf(`SELECT * FROM jobs WHERE %s = $1 LIMIT 1`, col)
		err = tx.Get(&jb, sql, arg)
		if err != nil {
			return errors.Wrap(err, "failed to load job")
		}

		if err := LoadAllJobTypes(tx, &jb); err != nil {
			return err
		}

		if jb.OffchainreportingOracleSpec != nil {
			var ch evm.Chain
			ch, err = o.chainSet.Get(jb.OffchainreportingOracleSpec.EVMChainID.ToInt())
			if err != nil {
				return err
			}
			jb.OffchainreportingOracleSpec = LoadDynamicConfigVars(ch.Config(), *jb.OffchainreportingOracleSpec)
		}
		return nil
	})
	return jb, err
}

func (o *orm) FindJobIDsWithBridge(name string) (jids []int32, err error) {
	err = postgres.SqlxTransactionWithDefaultCtx(o.db, func(tx *sqlx.Tx) error {
		sql := `SELECT jobs.id, dot_dag_source FROM jobs JOIN pipeline_specs ON pipeline_specs.id = jobs.pipeline_spec_id WHERE dot_dag_source ILIKE "%" || $1 || "%" ORDER BY id`
		rows, err := tx.Query(sql, name)
		if err != nil {
			return err
		}
		defer rows.Close()
		var ids []int32
		var sources []string
		for rows.Next() {
			var id int32
			var source string
			if err := rows.Scan(&id, &source); err != nil {
				return err
			}
			ids = append(jids, id)
			sources = append(sources, source)
		}

		for i, id := range ids {
			p, err := pipeline.Parse(sources[i])
			if err != nil {
				return errors.Wrapf(err, "could not parse dag for job %d", id)
			}
			for _, task := range p.Tasks {
				if task.Type() == pipeline.TaskTypeBridge {
					if task.(*pipeline.BridgeTask).Name == name {
						jids = append(jids, id)
					}
				}
			}
		}
		return nil
	})
	return jids, errors.Wrap(err, "FindJobIDsWithBridge failed")
}

// Preload PipelineSpec.JobID for each Run
func (o *orm) preloadJobIDs(runs []pipeline.Run) error {
	// Abort early if there are no runs
	if len(runs) == 0 {
		return nil
	}

	ids := make([]int32, 0, len(runs))
	for _, run := range runs {
		ids = append(ids, run.PipelineSpecID)
	}

	// construct a WHERE IN query
	sql := `SELECT id, pipeline_spec_id FROM jobs WHERE pipeline_spec_id IN (?);`
	query, args, err := sqlx.In(sql, ids)
	if err != nil {
		return err
	}
	query = o.db.Rebind(query)
	var results []struct {
		ID             int32
		PipelineSpecID int32
	}
	if err := o.db.Select(&results, query, args...); err != nil {
		return err
	}

	// fill in fields
	for i := range runs {
		for _, result := range results {
			if result.PipelineSpecID == runs[i].PipelineSpecID {
				runs[i].PipelineSpec.JobID = result.ID
			}
		}
	}

	return nil
}

// PipelineRuns returns pipeline runs for a job, with spec and taskruns loaded, latest first
// If jobID is nil, returns all pipeline runs
func (o *orm) PipelineRuns(jobID *int32, offset, size int) (runs []pipeline.Run, count int, err error) {
	err = postgres.SqlxTransactionWithDefaultCtx(o.db, func(tx *sqlx.Tx) error {
		var args []interface{}
		var where string
		if jobID != nil {
			where = " WHERE jobs.id = $1"
			args = append(args, *jobID)
		}
		sql := fmt.Sprintf(`SELECT count(*) FROM pipeline_runs INNER JOIN jobs ON pipeline_runs.pipeline_spec_id = jobs.pipeline_spec_id%s`, where)
		if err = tx.QueryRowx(sql, args...).Scan(&count); err != nil {
			return err
		}

		sql = fmt.Sprintf(`SELECT pipeline_runs.*, jobs.id AS job_id FROM pipeline_runs INNER JOIN jobs ON pipeline_runs.pipeline_spec_id = jobs.pipeline_spec_id%s
		ORDER BY pipeline_runs.created_at DESC, pipeline_runs.id DESC
		OFFSET $2 LIMIT $3
		;`, args...)

		if err = tx.Select(runs, sql, jobID, offset, size); err != nil {
			return err
		}

		// Postload PipelineSpecs
		// TODO: We should pull this out into a generic preload function once go has generics
		specM := make(map[int32]pipeline.Spec)
		for _, run := range runs {
			if _, exists := specM[run.PipelineSpecID]; !exists {
				specM[run.PipelineSpecID] = pipeline.Spec{}
			}
		}
		specIDs := make([]int32, len(specM))
		for specID := range specM {
			specIDs = append(specIDs, specID)
		}
		sql = `SELECT * FROM pipeline_specs WHERE id = ANY($1);`
		var specs []pipeline.Spec
		if err = o.db.Select(&specs, sql, specIDs); err != nil {
			return err
		}
		for _, spec := range specs {
			specM[spec.ID] = spec
		}
		runM := make(map[int64]*pipeline.Run, len(runs))
		for i, run := range runs {
			runs[i].PipelineSpec = specM[run.PipelineSpecID]
			runM[run.ID] = &runs[i]
		}

		// Postload PipelineTaskRuns
		runIDs := make([]int64, len(runs))
		for i, run := range runs {
			runIDs[i] = run.ID
		}
		var taskRuns []pipeline.TaskRun
		sql = `SELECT * FROM pipeline_task_runs WHERE pipeline_run_id = ANY($1) ORDER BY pipeline_run_id, created_at, id;`
		if err = tx.Select(&taskRuns, sql, runIDs); err != nil {
			return err
		}
		for _, taskRun := range taskRuns {
			run := runM[taskRun.PipelineRunID]
			run.PipelineTaskRuns = append(run.PipelineTaskRuns, taskRun)
		}
		return nil
	})

	return runs, count, err
}

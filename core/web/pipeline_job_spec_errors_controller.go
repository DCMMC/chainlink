package web

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/DCMMC/chainlink/core/services/chainlink"
	"github.com/DCMMC/chainlink/core/services/job"
	"gorm.io/gorm"
)

// PipelineJobSpecErrorsController manages PipelineJobSpecError requests
type PipelineJobSpecErrorsController struct {
	App chainlink.Application
}

// Destroy deletes a PipelineJobSpecError record from the database, effectively
// silencing the error notification
func (psec *PipelineJobSpecErrorsController) Destroy(c *gin.Context) {
	jobSpec := job.Job{}
	err := jobSpec.SetID(c.Param("ID"))
	if err != nil {
		jsonAPIError(c, http.StatusUnprocessableEntity, err)
		return
	}

	err = psec.App.JobORM().DismissError(context.Background(), jobSpec.ID)
	if errors.Cause(err) == gorm.ErrRecordNotFound {
		jsonAPIError(c, http.StatusNotFound, errors.New("PipelineJobSpecError not found"))
		return
	}
	if err != nil {
		jsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	jsonAPIResponseWithStatus(c, nil, "job", http.StatusNoContent)
}

package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/DCMMC/chainlink/core/utils"
	"github.com/DCMMC/chainlink/core/web/presenters"
	"github.com/urfave/cli"
	"go.uber.org/multierr"
)

type EthKeyPresenter struct {
	presenters.ETHKeyResource
}

func (p *EthKeyPresenter) ToRow() []string {
	return []string{
		p.Address,
		p.EVMChainID.String(),
		p.EthBalance.String(),
		p.LinkBalance.String(),
		fmt.Sprintf("%v", p.IsFunding),
		p.CreatedAt.String(),
		p.UpdatedAt.String(),
	}
}

// RenderTable implements TableRenderer
func (p *EthKeyPresenter) RenderTable(rt RendererTable) error {
	headers := []string{"Address", "EVM Chain ID", "ETH", "LINK", "Is funding", "Created", "Updated"}
	rows := [][]string{p.ToRow()}

	renderList(headers, rows, rt.Writer)

	return utils.JustError(rt.Write([]byte("\n")))
}

type EthKeyPresenters []EthKeyPresenter

// RenderTable implements TableRenderer
func (ps EthKeyPresenters) RenderTable(rt RendererTable) error {
	headers := []string{"Address", "EVM Chain ID", "ETH", "LINK", "Is funding", "Created", "Updated"}
	rows := [][]string{}

	for _, p := range ps {
		rows = append(rows, p.ToRow())
	}

	renderList(headers, rows, rt.Writer)

	return nil
}

// ListETHKeys renders the active account address with its ETH & LINK balance
func (cli *Client) ListETHKeys(c *cli.Context) (err error) {
	resp, err := cli.HTTP.Get("/v2/keys/eth")
	if err != nil {
		return cli.errorOut(err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			err = multierr.Append(err, cerr)
		}
	}()

	return cli.renderAPIResponse(resp, &EthKeyPresenters{}, "🔑 ETH keys")
}

// CreateETHKey creates a new ethereum key with the same password
// as the one used to unlock the existing key.
func (cli *Client) CreateETHKey(c *cli.Context) (err error) {
	query := url.Values{}
	if c.IsSet("evmChainID") {
		query.Set("evmChainID", c.String("evmChainID"))
	}
	resp, err := cli.HTTP.Post("/v2/keys/eth?"+query.Encode(), nil)
	if err != nil {
		return cli.errorOut(err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			err = multierr.Append(err, cerr)
		}
	}()

	return cli.renderAPIResponse(resp, &EthKeyPresenter{}, "ETH key created.\n\n🔑 New key")
}

// DeleteETHKey deletes an Ethereum key,
// address of key must be passed
func (cli *Client) DeleteETHKey(c *cli.Context) (err error) {
	if !c.Args().Present() {
		return cli.errorOut(errors.New("Must pass the address of the key to be deleted"))
	}

	if c.Bool("hard") && !confirmAction(c) {
		return nil
	}

	var queryStr string
	var confirmationMsg string
	if c.Bool("hard") {
		queryStr = "?hard=true"
		confirmationMsg = "Deleted ETH key"
	} else {
		confirmationMsg = "Archived ETH key"
	}

	address := c.Args().Get(0)
	resp, err := cli.HTTP.Delete(fmt.Sprintf("/v2/keys/eth/%s%s", address, queryStr))
	if err != nil {
		return cli.errorOut(err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			err = multierr.Append(err, cerr)
		}
	}()

	return cli.renderAPIResponse(resp, &EthKeyPresenter{}, fmt.Sprintf("🔑 %s", confirmationMsg))
}

// ImportETHKey imports an Ethereum key,
// file path must be passed
func (cli *Client) ImportETHKey(c *cli.Context) (err error) {
	if !c.Args().Present() {
		return cli.errorOut(errors.New("Must pass the filepath of the key to be imported"))
	}

	oldPasswordFile := c.String("oldpassword")
	if len(oldPasswordFile) == 0 {
		return cli.errorOut(errors.New("Must specify --oldpassword/-p flag"))
	}
	oldPassword, err := ioutil.ReadFile(oldPasswordFile)
	if err != nil {
		return cli.errorOut(errors.Wrap(err, "Could not read password file"))
	}

	filepath := c.Args().Get(0)
	keyJSON, err := ioutil.ReadFile(filepath)
	if err != nil {
		return cli.errorOut(err)
	}

	query := url.Values{}
	query.Set("oldpassword", strings.TrimSpace(string(oldPassword)))

	if c.IsSet("evmChainID") {
		query.Set("evmChainID", c.String("evmChainID"))
	}

	resp, err := cli.HTTP.Post("/v2/keys/eth/import?"+query.Encode(), bytes.NewReader(keyJSON))
	if err != nil {
		return cli.errorOut(err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			err = multierr.Append(err, cerr)
		}
	}()

	return cli.renderAPIResponse(resp, &EthKeyPresenter{}, "🔑 Imported ETH key")
}

// ExportETHKey exports an ETH key,
// address must be passed
func (cli *Client) ExportETHKey(c *cli.Context) (err error) {
	if !c.Args().Present() {
		return cli.errorOut(errors.New("Must pass the address of the key to export"))
	}

	newPasswordFile := c.String("newpassword")
	if len(newPasswordFile) == 0 {
		return cli.errorOut(errors.New("Must specify --newpassword/-p flag"))
	}
	newPassword, err := ioutil.ReadFile(newPasswordFile)
	if err != nil {
		return cli.errorOut(errors.Wrap(err, "Could not read password file"))
	}

	filepath := c.String("output")
	if len(newPassword) == 0 {
		return cli.errorOut(errors.New("Must specify --output/-o flag"))
	}

	address := c.Args().Get(0)

	query := url.Values{}
	query.Set("newpassword", strings.TrimSpace(string(newPassword)))

	resp, err := cli.HTTP.Post("/v2/keys/eth/export/"+address+"?"+query.Encode(), nil)
	if err != nil {
		return cli.errorOut(errors.Wrap(err, "Could not make HTTP request"))
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			err = multierr.Append(err, cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return cli.errorOut(errors.New("Error exporting"))
	}

	keyJSON, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return cli.errorOut(errors.Wrap(err, "Could not read response body"))
	}

	err = utils.WriteFileWithMaxPerms(filepath, keyJSON, 0600)
	if err != nil {
		return cli.errorOut(errors.Wrapf(err, "Could not write %v", filepath))
	}

	_, err = os.Stderr.WriteString("🔑 Exported ETH key " + address + " to " + filepath + "\n")
	if err != nil {
		return cli.errorOut(err)
	}

	return nil
}

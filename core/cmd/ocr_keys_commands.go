package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/pkg/errors"
	"github.com/DCMMC/chainlink/core/store/models"
	"github.com/DCMMC/chainlink/core/utils"
	"github.com/DCMMC/chainlink/core/web/presenters"
	"github.com/urfave/cli"
	"go.uber.org/multierr"
)

type OCRKeyBundlePresenter struct {
	JAID // Include this to overwrite the presenter JAID so it can correctly render the ID in JSON
	presenters.OCRKeysBundleResource
}

// RenderTable implements TableRenderer
func (p *OCRKeyBundlePresenter) RenderTable(rt RendererTable) error {
	headers := []string{"ID", "On-chain signing addr", "Off-chain pubkey", "Config pubkey"}
	rows := [][]string{p.ToRow()}

	if _, err := rt.Write([]byte("🔑 OCR Keys\n")); err != nil {
		return err
	}
	renderList(headers, rows, rt.Writer)

	return utils.JustError(rt.Write([]byte("\n")))
}

func (p *OCRKeyBundlePresenter) ToRow() []string {
	return []string{
		p.ID,
		p.OnChainSigningAddress.String(),
		p.OffChainPublicKey.String(),
		p.ConfigPublicKey.String(),
	}
}

type OCRKeyBundlePresenters []OCRKeyBundlePresenter

// RenderTable implements TableRenderer
func (ps OCRKeyBundlePresenters) RenderTable(rt RendererTable) error {
	headers := []string{"ID", "On-chain signing addr", "Off-chain pubkey", "Config pubkey"}
	rows := [][]string{}

	for _, p := range ps {
		rows = append(rows, p.ToRow())
	}

	if _, err := rt.Write([]byte("🔑 OCR Keys\n")); err != nil {
		return err
	}
	renderList(headers, rows, rt.Writer)

	return utils.JustError(rt.Write([]byte("\n")))
}

// ListOCRKeyBundles lists the available OCR Key Bundles
func (cli *Client) ListOCRKeyBundles(c *cli.Context) error {
	resp, err := cli.HTTP.Get("/v2/keys/ocr", nil)
	if err != nil {
		return cli.errorOut(err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			err = multierr.Append(err, cerr)
		}
	}()

	var presenters OCRKeyBundlePresenters
	return cli.renderAPIResponse(resp, &presenters)
}

// CreateOCRKeyBundle creates a key and inserts it into encrypted_ocr_key_bundles,
// protected by the password in the password file
func (cli *Client) CreateOCRKeyBundle(c *cli.Context) error {
	resp, err := cli.HTTP.Post("/v2/keys/ocr", nil)
	if err != nil {
		return cli.errorOut(err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			err = multierr.Append(err, cerr)
		}
	}()

	var presenter OCRKeyBundlePresenter
	return cli.renderAPIResponse(resp, &presenter, "Created OCR key bundle")
}

// DeleteOCRKeyBundle creates a key and inserts it into encrypted_ocr_keys,
// protected by the password in the password file
func (cli *Client) DeleteOCRKeyBundle(c *cli.Context) error {
	if !c.Args().Present() {
		return cli.errorOut(errors.New("Must pass the key ID to be deleted"))
	}
	id, err := models.Sha256HashFromHex(c.Args().Get(0))
	if err != nil {
		return cli.errorOut(err)
	}

	if !confirmAction(c) {
		return nil
	}

	var queryStr string
	if c.Bool("hard") {
		queryStr = "?hard=true"
	}

	resp, err := cli.HTTP.Delete(fmt.Sprintf("/v2/keys/ocr/%s%s", id, queryStr))
	if err != nil {
		return cli.errorOut(err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			err = multierr.Append(err, cerr)
		}
	}()

	var presenter OCRKeyBundlePresenter
	return cli.renderAPIResponse(resp, &presenter, "OCR key bundle deleted")
}

// ImportOCRKey imports OCR key bundle,
// file path must be passed
func (cli *Client) ImportOCRKey(c *cli.Context) (err error) {
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

	normalizedPassword := normalizePassword(string(oldPassword))
	resp, err := cli.HTTP.Post("/v2/keys/ocr/import?oldpassword="+normalizedPassword, bytes.NewReader(keyJSON))
	if err != nil {
		return cli.errorOut(err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			err = multierr.Append(err, cerr)
		}
	}()

	var presenter OCRKeyBundlePresenter
	return cli.renderAPIResponse(resp, &presenter, "Imported OCR key bundle")
}

// ExportOCRKey exports OCR key bundles by ID
// ID of the key must be passed
func (cli *Client) ExportOCRKey(c *cli.Context) (err error) {
	if !c.Args().Present() {
		return cli.errorOut(errors.New("Must pass the ID of the key to export"))
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
	if len(filepath) == 0 {
		return cli.errorOut(errors.New("Must specify --output/-o flag"))
	}

	ID := c.Args().Get(0)

	normalizedPassword := normalizePassword(string(newPassword))
	resp, err := cli.HTTP.Post("/v2/keys/ocr/export/"+ID+"?newpassword="+normalizedPassword, nil)
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

	_, err = os.Stderr.WriteString(fmt.Sprintf("Exported OCR key bundle %s to %s", ID, filepath))
	if err != nil {
		return cli.errorOut(err)
	}

	return nil
}

package shape

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/KogenAI/kogen-coordinator/project"
)

// A Shape draft is an Intent whose slug has a shaping record directory
// `.kogen/local/shaping/<slug>/`. It is approvable only when Shape (or Refine) recorded that
// its checks passed for exactly the current package and the request of the source receipt.
// The record is `.kogen/local/shaping/<slug>/passed.json`; an Intent without the shaping
// directory (written by hand) has no draft and needs no record.

type shapePass struct {
	PackageSHA256  string `json:"package_sha256"`
	RequestSHA256  string `json:"request_sha256"`
	ContractReview bool   `json:"contract_review"`
	Readiness      bool   `json:"readiness"`
}

func shapePassPath(root, slug string) string {
	return filepath.Join(root, ".kogen", "local", "shaping", slug, "passed.json")
}

func shapingRecordDir(root, slug string) bool {
	fi, err := os.Stat(filepath.Join(root, ".kogen", "local", "shaping", slug))
	return err == nil && fi.IsDir()
}

// receiptRequestDigest is the digest of the request text the source receipt captured; empty
// when the Intent has no receipt or its request cannot be read.
func receiptRequestDigest(root, slug string) string {
	b, err := os.ReadFile(project.SourcePath(root, slug))
	if err != nil {
		return ""
	}
	var doc struct {
		Request struct {
			Text string `json:"text"`
		} `json:"request"`
	}
	if json.Unmarshal(b, &doc) != nil {
		return ""
	}
	return ContentHash([]byte(doc.Request.Text))
}

func currentPackageHash(root, slug string, cfg *project.Config) (string, error) {
	paths, err := project.Baseline(root, slug, cfg)
	if err != nil {
		return "", err
	}
	return project.Hash(root, paths), nil
}

// recordShapePass records that the checks passed for the current package and the receipt's
// request.
func recordShapePass(root, slug string, cfg *project.Config) error {
	hash, err := currentPackageHash(root, slug, cfg)
	if err != nil {
		return err
	}
	data, err := json.Marshal(shapePass{PackageSHA256: hash, RequestSHA256: receiptRequestDigest(root, slug), ContractReview: true, Readiness: true})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(shapePassPath(root, slug)), 0o755); err != nil {
		return err
	}
	return os.WriteFile(shapePassPath(root, slug), data, 0o600)
}

// refinePass records the pass of a Refine only where a shaping record directory already
// makes the Intent a Shape draft; a hand-written Intent stays without one.
func refinePass(root, slug string, cfg *project.Config) error {
	if !shapingRecordDir(root, slug) {
		return nil
	}
	return recordShapePass(root, slug, cfg)
}

func clearShapePass(root, slug string) error {
	if err := os.Remove(shapePassPath(root, slug)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// unmatchedPass reports whether the current package hash and receipt request do not match a
// recorded pass; callers apply it to Shape drafts only. A package that cannot be hashed is
// not judged here: the callers' own checks report it.
func unmatchedPass(root, slug string, cfg *project.Config) bool {
	hash, err := currentPackageHash(root, slug, cfg)
	if err != nil {
		return false
	}
	b, err := os.ReadFile(shapePassPath(root, slug))
	if err != nil {
		return true
	}
	var pass shapePass
	if json.Unmarshal(b, &pass) != nil {
		return true
	}
	return !(pass.ContractReview && pass.Readiness && pass.PackageSHA256 == hash && pass.RequestSHA256 == receiptRequestDigest(root, slug))
}

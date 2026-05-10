package converter

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func HTMLToPDF(ctx context.Context, html []byte) ([]byte, error) {
	dir, err := os.MkdirTemp("", "pdf-converter-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(dir)

	htmlPath := filepath.Join(dir, "input.html")
	pdfPath := filepath.Join(dir, "output.pdf")

	if err := os.WriteFile(htmlPath, html, 0644); err != nil {
		return nil, fmt.Errorf("write html: %w", err)
	}

	cmd := exec.CommandContext(ctx, "wkhtmltopdf",
		"--quiet",
		"--disable-smart-shrinking",
		"--load-error-handling", "ignore",
		"--load-media-error-handling", "ignore",
		"--encoding", "utf-8",
		htmlPath, pdfPath,
	)

	if res, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("wkhtmltopdf error: %w, output: %s", err, string(res))
	}

	pdf, err := os.ReadFile(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("read pdf: %w", err)
	}

	return pdf, nil
}

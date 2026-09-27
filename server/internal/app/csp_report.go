package app

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// cspReport is the part of a browser's report-uri body worth logging.
type cspReport struct {
	Report struct {
		DocumentURI        string `json:"document-uri"`
		ViolatedDirective  string `json:"violated-directive"`
		EffectiveDirective string `json:"effective-directive"`
		BlockedURI         string `json:"blocked-uri"`
	} `json:"csp-report"`
}

// maxReportField keeps one noisy page from filling the logs.
const maxReportField = 200

// cspReportHandler logs the web app's Content-Security-Policy violations, so
// the report-only policy can be tuned before it is enforced (ADR 0022).
// Browsers send these without credentials, so it needs no token.
func cspReportHandler(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var report cspReport
		if err := json.NewDecoder(c.Request.Body).Decode(&report); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, errorResponse("VALIDATION_ERROR", "invalid CSP report"))
			return
		}
		logger.Warn("content security policy violation",
			"document_uri", truncate(report.Report.DocumentURI),
			"violated_directive", truncate(report.Report.ViolatedDirective),
			"effective_directive", truncate(report.Report.EffectiveDirective),
			"blocked_uri", truncate(report.Report.BlockedURI),
		)
		c.Status(http.StatusNoContent)
	}
}

func truncate(value string) string {
	if len(value) > maxReportField {
		return value[:maxReportField]
	}
	return value
}

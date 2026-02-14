package maintenance

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/memory"
	providerpkg "github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
)

const (
	dailyFileAgeDays         = 3
	compostingAgeMonths      = 6
	llmTemperature           = 0.3
	llmMaxTokens             = 500
	defaultInsightImportance = 0.7
	minFileContentLength     = 50
)

type NotificationEmitter interface {
	NotifyMemoryCompression(filesProcessed int, compressionRatio float64)
	NotifySeedsExtracted(seedCount int, sessionID string)
}

type MemoryMaintenance struct {
	memoryManager      *memory.MemoryManager
	compressionService *memory.CompressionService
	pluginManager      memory.PluginManagerInterface
	memoryDir          string
	archiveDir         string
	memoryFile         string
	mu                 sync.Mutex
	notifier           NotificationEmitter
}

func NewMemoryMaintenance(memoryManager *memory.MemoryManager, pluginManager memory.PluginManagerInterface, memoryDir string) (*MemoryMaintenance, error) {
	if memoryManager == nil {
		return nil, fmt.Errorf("memory manager cannot be nil")
	}
	if pluginManager == nil {
		return nil, fmt.Errorf("plugin manager cannot be nil")
	}
	if memoryDir == "" {
		memoryDir = "memory"
	}

	archiveDir := filepath.Join(memoryDir, "archive")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create archive directory: %w", err)
	}

	return &MemoryMaintenance{
		memoryManager:      memoryManager,
		compressionService: memory.NewCompressionService(pluginManager),
		pluginManager:      pluginManager,
		memoryDir:          memoryDir,
		archiveDir:         archiveDir,
		memoryFile:         filepath.Join(memoryDir, "MEMORY.md"),
		notifier:           nil,
	}, nil
}

func (mm *MemoryMaintenance) SetNotifier(notifier NotificationEmitter) {
	mm.notifier = notifier
}

func (mm *MemoryMaintenance) RunMaintenance(ctx context.Context) (*MaintenanceReport, error) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	report := &MaintenanceReport{
		StartTime: time.Now(),
		Errors:    []string{},
		Success:   true,
	}

	beforeSize, err := mm.getMemorySize()
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("failed to get initial memory size: %v", err))
	}
	report.MemorySize.BeforeBytes = beforeSize

	slog.Info("starting memory maintenance cycle")

	dailyFiles, err := mm.scanDailyFiles(ctx)
	if err != nil {
		report.Success = false
		report.Errors = append(report.Errors, fmt.Sprintf("failed to scan daily files: %v", err))
		mm.finalizeReport(report)
		return report, fmt.Errorf("failed to scan daily files: %w", err)
	}

	slog.Info("found daily files to process", "count", len(dailyFiles))
	report.FilesProcessed = len(dailyFiles)

	insights := []Insight{}
	for _, filePath := range dailyFiles {
		fileInsights, err := mm.IdentifyInsights(ctx, filePath)
		if err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("failed to extract insights from %s: %v", filePath, err))
			continue
		}
		insights = append(insights, fileInsights...)
	}

	slog.Info("extracted insights from daily files", "count", len(insights))
	report.InsightsExtracted = len(insights)

	if len(insights) > 0 {
		if err := mm.AppendToMemory(ctx, insights); err != nil {
			report.Success = false
			report.Errors = append(report.Errors, fmt.Sprintf("failed to append insights to MEMORY.md: %v", err))
		}
	}

	for _, filePath := range dailyFiles {
		if err := mm.ArchiveFile(ctx, filePath); err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("failed to archive %s: %v", filePath, err))
			continue
		}
		report.FilesArchived++
	}

	slog.Info("archived files", "count", report.FilesArchived)

	composted, err := mm.CompostMemory(ctx)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("failed to compost memory: %v", err))
	} else {
		report.EntriesComposted = composted
		slog.Info("composted outdated entries", "count", composted)
	}

	backfillService := memory.NewBackfillService(mm.memoryManager, nil)
	backfillStatus, err := backfillService.GetBackfillStatus(ctx)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("failed to get backfill status: %v", err))
	} else {
		report.BackfillProgress = backfillStatus.ProgressPercent
		report.EmbeddingsRemaining = backfillStatus.TotalMemories - backfillStatus.MemoriesWithEmbedding

		if report.EmbeddingsRemaining > 0 {
			slog.Info("running embedding backfill", "progress_percent", backfillStatus.ProgressPercent)
			backfillResult, err := backfillService.ProcessBackfill(ctx, 1000, false, false)
			if err != nil {
				report.Errors = append(report.Errors, fmt.Sprintf("backfill failed: %v", err))
			} else {
				report.EmbeddingsBackfilled = backfillResult.SuccessCount
				slog.Info("backfilled embeddings", "count", backfillResult.SuccessCount, "duration", backfillResult.Duration)

				if backfillResult.FailedCount > 0 {
					report.Errors = append(report.Errors, fmt.Sprintf("backfill: %d failures", backfillResult.FailedCount))
				}
			}
		} else {
			slog.Info("all memories have embeddings")
		}
	}

	afterSize, err := mm.getMemorySize()
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("failed to get final memory size: %v", err))
	}
	report.MemorySize.AfterBytes = afterSize
	report.MemorySize.Reduction = beforeSize - afterSize

	mm.finalizeReport(report)

	slog.Info("memory maintenance completed",
		"duration", report.Duration,
		"files_processed", report.FilesProcessed,
		"files_archived", report.FilesArchived,
		"insights_extracted", report.InsightsExtracted,
		"entries_composted", report.EntriesComposted,
		"embeddings_backfilled", report.EmbeddingsBackfilled,
	)

	if len(report.Errors) > 0 {
		slog.Warn("maintenance completed with errors", "error_count", len(report.Errors))
	}

	if mm.notifier != nil && report.FilesArchived > 0 {
		compressionRatio := 0.0
		if beforeSize > 0 {
			compressionRatio = float64(report.MemorySize.Reduction) / float64(beforeSize)
		}
		mm.notifier.NotifyMemoryCompression(report.FilesArchived, compressionRatio)
	}

	return report, nil
}

func (mm *MemoryMaintenance) scanDailyFiles(ctx context.Context) ([]string, error) {
	cutoffDate := time.Now().AddDate(0, 0, -dailyFileAgeDays)

	entries, err := os.ReadDir(mm.memoryDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read memory directory: %w", err)
	}

	var dailyFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".md") || name == "MEMORY.md" {
			continue
		}

		if len(name) < 10 {
			continue
		}

		datePart := strings.TrimSuffix(name, ".md")
		fileDate, err := time.Parse("2006-01-02", datePart)
		if err != nil {
			continue
		}

		if fileDate.Before(cutoffDate) {
			dailyFiles = append(dailyFiles, filepath.Join(mm.memoryDir, name))
		}
	}

	return dailyFiles, nil
}

func (mm *MemoryMaintenance) IdentifyInsights(ctx context.Context, dailyFile string) ([]Insight, error) {
	content, err := os.ReadFile(dailyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	contentStr := string(content)
	if len(contentStr) < minFileContentLength {
		return []Insight{}, nil
	}

	themes, err := mm.compressionService.IdentifyKeyThemes(ctx, contentStr)
	if err != nil {
		return nil, fmt.Errorf("failed to identify themes: %w", err)
	}

	provIface, err := mm.pluginManager.GetProvider("embedded-qwen3")
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	prov, ok := provIface.(providerpkg.ModelProvider)
	if !ok {
		return nil, fmt.Errorf("provider does not implement ModelProvider interface")
	}

	prompt := mm.buildInsightExtractionPrompt(contentStr, themes)

	req := &providerpkg.CompletionRequest{
		Model: "embedded-qwen3",
		Messages: []providerpkg.Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: llmTemperature,
		MaxTokens:   llmMaxTokens,
		Stream:      false,
	}

	resp, err := prov.GenerateCompletion(ctx, req)
	if err != nil {
		slog.Error("failed to generate insights",
			"component", "memory_maintenance",
			"method", "IdentifyInsights",
			"error", err,
			"model", "embedded-qwen3",
			"file", dailyFile,
			"content_length", len(contentStr),
		)
		return nil, fmt.Errorf("failed to generate insights: %w", err)
	}

	insights := mm.parseInsights(resp.Content, themes, dailyFile)

	return insights, nil
}

func (mm *MemoryMaintenance) buildInsightExtractionPrompt(content string, themes []string) string {
	themesStr := strings.Join(themes, ", ")
	return fmt.Sprintf(`Analyze the following daily memory content and extract 2-3 key insights that would be valuable to remember long-term.

Themes identified: %s

For each insight, provide:
1. A one-sentence summary (under 100 characters)
2. Why it matters (1-2 sentences)

Focus on:
- Key decisions and their rationale
- Important patterns or lessons learned
- Actionable knowledge that would matter in 3+ months
- Technical insights or discoveries

Exclude:
- Greetings, confirmations, or pleasantries
- Routine tasks without novel insights
- Temporary context or session-specific details

Daily content:
%s

Format each insight as:
INSIGHT: [one-sentence summary]
WHY: [why it matters]
---`, themesStr, content)
}

func (mm *MemoryMaintenance) parseInsights(response string, themes []string, source string) []Insight {
	insights := []Insight{}
	now := time.Now()

	blocks := strings.Split(response, "---")
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}

		lines := strings.Split(block, "\n")
		var summary, why string

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "INSIGHT:") {
				summary = strings.TrimSpace(strings.TrimPrefix(line, "INSIGHT:"))
			} else if strings.HasPrefix(line, "WHY:") {
				why = strings.TrimSpace(strings.TrimPrefix(line, "WHY:"))
			}
		}

		if summary != "" && why != "" {
			theme := ""
			if len(themes) > 0 {
				theme = themes[0]
			}

			insights = append(insights, Insight{
				Theme:       theme,
				Summary:     summary,
				Source:      filepath.Base(source),
				Importance:  defaultInsightImportance,
				ExtractedAt: now,
			})
		}
	}

	return insights
}

func (mm *MemoryMaintenance) AppendToMemory(ctx context.Context, insights []Insight) error {
	if len(insights) == 0 {
		return nil
	}

	var content strings.Builder

	if _, err := os.Stat(mm.memoryFile); err == nil {
		existingContent, err := os.ReadFile(mm.memoryFile)
		if err != nil {
			return fmt.Errorf("failed to read existing MEMORY.md: %w", err)
		}
		content.Write(existingContent)
		content.WriteString("\n\n")
	}

	content.WriteString(fmt.Sprintf("## Memory Maintenance - %s\n\n", time.Now().Format("2006-01-02")))

	for _, insight := range insights {
		content.WriteString(fmt.Sprintf("- **%s** (%s)\n", insight.Summary, insight.Source))
	}

	if err := os.WriteFile(mm.memoryFile, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("failed to write MEMORY.md: %w", err)
	}

	return nil
}

func (mm *MemoryMaintenance) ArchiveFile(ctx context.Context, filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	contentStr := string(content)
	compressed, err := mm.compressionService.CompressHistory(ctx, filepath.Base(filePath), []memory.Memory{
		{
			ID:      filepath.Base(filePath),
			Type:    "daily_note",
			Content: contentStr,
		},
	})
	if err != nil {
		slog.Warn("failed to compress file, archiving uncompressed", "file", filePath, "error", err)
	} else {
		contentStr = compressed.CompressedContent
	}

	archivePath := filepath.Join(mm.archiveDir, filepath.Base(filePath))
	if err := os.WriteFile(archivePath, []byte(contentStr), 0644); err != nil {
		return fmt.Errorf("failed to write archive file: %w", err)
	}

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to remove original file: %w", err)
	}

	return nil
}

func (mm *MemoryMaintenance) CompostMemory(ctx context.Context) (int, error) {
	if _, err := os.Stat(mm.memoryFile); os.IsNotExist(err) {
		return 0, nil
	}

	content, err := os.ReadFile(mm.memoryFile)
	if err != nil {
		return 0, fmt.Errorf("failed to read MEMORY.md: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	filtered := []string{}
	composted := 0

	cutoffDate := time.Now().AddDate(0, -compostingAgeMonths, 0)

	inOldSection := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "## Memory Maintenance - ") {
			dateStr := strings.TrimPrefix(trimmed, "## Memory Maintenance - ")
			if entryDate, err := time.Parse("2006-01-02", dateStr); err == nil {
				if entryDate.Before(cutoffDate) {
					composted++
					inOldSection = true
					continue
				} else {
					inOldSection = false
				}
			} else {
				inOldSection = false
			}
		} else if strings.HasPrefix(trimmed, "##") || strings.HasPrefix(trimmed, "#") {
			inOldSection = false
		}

		if inOldSection {
			if i+1 < len(lines) {
				nextTrimmed := strings.TrimSpace(lines[i+1])
				if strings.HasPrefix(nextTrimmed, "##") || strings.HasPrefix(nextTrimmed, "#") {
					inOldSection = false
				}
			}
			continue
		}

		filtered = append(filtered, line)
	}

	if composted > 0 {
		newContent := strings.Join(filtered, "\n")
		if err := os.WriteFile(mm.memoryFile, []byte(newContent), 0644); err != nil {
			return 0, fmt.Errorf("failed to write composted MEMORY.md: %w", err)
		}
	}

	return composted, nil
}

func (mm *MemoryMaintenance) getMemorySize() (int64, error) {
	var totalSize int64

	err := filepath.Walk(mm.memoryDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil {
		return 0, err
	}

	return totalSize, nil
}

func (mm *MemoryMaintenance) finalizeReport(report *MaintenanceReport) {
	report.EndTime = time.Now()
	report.Duration = report.EndTime.Sub(report.StartTime)

	if len(report.Errors) > 0 {
		report.Success = false
	}
}

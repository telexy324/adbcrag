package service

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"ops-kb-rag/backend/internal/util"
)

type TextChunk struct {
	Index         int
	Content       string
	SourceTitle   string
	SourceSection string
	TokenCount    int
}

type ChunkService struct {
	chunkSize    int
	chunkOverlap int
}

func NewChunkService() *ChunkService {
	return &ChunkService{chunkSize: 800, chunkOverlap: 100}
}

func (s *ChunkService) Split(title, text string) []TextChunk {
	text = util.NormalizeText(text)
	sections := splitMarkdownSections(text)
	if len(sections) == 0 {
		sections = splitParagraphSections(text)
	}
	var chunks []TextChunk
	for _, section := range sections {
		for _, part := range s.splitLongText(section.content) {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			chunks = append(chunks, TextChunk{
				Index:         len(chunks),
				Content:       part,
				SourceTitle:   title,
				SourceSection: section.title,
				TokenCount:    util.EstimateTokenCount(part),
			})
		}
	}
	return chunks
}

type sectionText struct {
	title   string
	content string
}

func splitMarkdownSections(text string) []sectionText {
	re := regexp.MustCompile(`(?m)^#{1,3}\s+.+$`)
	locs := re.FindAllStringIndex(text, -1)
	if len(locs) == 0 {
		return nil
	}
	sections := make([]sectionText, 0, len(locs))
	for i, loc := range locs {
		end := len(text)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		block := strings.TrimSpace(text[loc[0]:end])
		title := strings.TrimSpace(strings.TrimLeft(strings.SplitN(block, "\n", 2)[0], "# "))
		sections = append(sections, sectionText{title: title, content: block})
	}
	return sections
}

func splitParagraphSections(text string) []sectionText {
	parts := regexp.MustCompile(`\n\s*\n`).Split(text, -1)
	sections := make([]sectionText, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			sections = append(sections, sectionText{content: part})
		}
	}
	return sections
}

func (s *ChunkService) splitLongText(text string) []string {
	if utf8.RuneCountInString(text) <= s.chunkSize {
		return []string{text}
	}
	runes := []rune(text)
	var chunks []string
	for start := 0; start < len(runes); {
		end := start + s.chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
		if end == len(runes) {
			break
		}
		start = end - s.chunkOverlap
		if start < 0 {
			start = 0
		}
	}
	return chunks
}

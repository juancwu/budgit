package handler

import (
	"log/slog"

	"git.juancwu.dev/juancwu/budgit/internal/service"
)

// processTagNames normalizes tag names, deduplicates them, and resolves them
// to tag IDs. Tags that don't exist are auto-created.
func processTagNames(tagService *service.TagService, spaceID string, tagNames []string) ([]string, error) {
	existingTags, err := tagService.GetTagsForSpace(spaceID)
	if err != nil {
		return nil, err
	}

	existingTagsMap := make(map[string]string)
	for _, t := range existingTags {
		existingTagsMap[t.Name] = t.ID
	}

	var finalTagIDs []string
	processedTags := make(map[string]bool)

	for _, rawTagName := range tagNames {
		tagName := service.NormalizeTagName(rawTagName)
		if tagName == "" {
			continue
		}
		if processedTags[tagName] {
			continue
		}

		if id, exists := existingTagsMap[tagName]; exists {
			finalTagIDs = append(finalTagIDs, id)
		} else {
			newTag, err := tagService.CreateTag(spaceID, tagName, nil)
			if err != nil {
				slog.Error("failed to create new tag", "error", err, "tag_name", tagName)
				continue
			}
			finalTagIDs = append(finalTagIDs, newTag.ID)
			existingTagsMap[tagName] = newTag.ID
		}
		processedTags[tagName] = true
	}

	return finalTagIDs, nil
}

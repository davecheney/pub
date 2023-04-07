package workers

import (
	"bufio"
	"context"
	"fmt"
	"image"
	"net/http"
	"time"

	"github.com/davecheney/pub/internal/algorithms"
	"github.com/davecheney/pub/models"
	"gorm.io/gorm"
)

func NewStatusAttachmentRequestProcessor(db *gorm.DB) func(context.Context) error {
	return func(ctx context.Context) error {
		fmt.Println("StatusAttachmentRequestProcessor started")
		defer fmt.Println("StatusAttachmentRequestProcessor stopped")

		db := db.WithContext(ctx)
		for {
			if err := process(db, statusAttachementRequestScope, processStatusAttachmentRequest); err != nil {
				return err
			}
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(30 * time.Second):
				// continue
			}
		}
	}
}

func statusAttachementRequestScope(db *gorm.DB) *gorm.DB {
	return db.Preload("StatusAttachment").Where("attempts < 3")
}

func processStatusAttachmentRequest(tx *gorm.DB, request *models.StatusAttachmentRequest) error {
	fmt.Println("StatusAttachmentRequestProcessor", request.StatusAttachment.URL)
	ctx, cancel := context.WithTimeout(tx.Statement.Context, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, request.StatusAttachment.URL, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	headerContentType := resp.Header.Get("Content-Type")

	// read first 512 bytes to check the content type
	br := bufio.NewReader(resp.Body)
	head, err := br.Peek(512)
	if err != nil {
		return err
	}
	contentType := http.DetectContentType(head)
	if !algorithms.Equal(headerContentType, contentType, request.StatusAttachment.MediaType) {
		fmt.Println("StatusAttachmentRequestProcessor", request.StatusAttachment.URL, "content type mismatch, header:", headerContentType, "detected:", contentType, "db:", request.StatusAttachment.MediaType)
	}

	img, format, err := image.Decode(br)
	if err != nil {
		return err
	}
	b := img.Bounds()
	fmt.Println("StatusAttachmentRequestProcessor", request.StatusAttachment.URL, "format", format, "bounds", b)
	return tx.Model(request.StatusAttachment).
		Updates(map[string]interface{}{
			"media_type": contentType,
			"width":      b.Dx(),
			"height":     b.Dy(),
		}).Error
}

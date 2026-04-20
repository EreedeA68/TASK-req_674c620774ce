package documents

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"

	"github.com/keystone/backend/internal/db"
	"gorm.io/gorm"
)

// AuditLogger is the unified audit interface for document events.
type AuditLogger interface {
	Log(actorID, action, resourceType, resourceID, deviceID, ip string, before, after interface{}) error
	CheckDownloadPermission(userID, resourceType, resourceID string) (bool, error)
}

// Service implements document management logic.
type Service struct {
	db    *gorm.DB
	audit AuditLogger
}

// NewService creates a new documents Service.
func NewService(database *gorm.DB, audit AuditLogger) *Service {
	return &Service{db: database, audit: audit}
}

// getDocument retrieves a CandidateDocument by ID.
func (s *Service) getDocument(documentID string) (*db.CandidateDocument, error) {
	var doc db.CandidateDocument
	if err := s.db.Where("id = ?", documentID).First(&doc).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("document not found")
		}
		return nil, err
	}
	return &doc, nil
}

// GenerateDownloadURL checks permission and returns a signed download path.
func (s *Service) GenerateDownloadURL(documentID, userID string) (string, error) {
	ok, err := s.audit.CheckDownloadPermission(userID, "candidate_document", documentID)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", errors.New("download not permitted")
	}
	return "/api/documents/" + documentID + "/download", nil
}

// DownloadDocument checks permission, reads the file from disk, applies watermark if enabled,
// logs the download via the unified audit schema, and returns the file bytes and original filename.
func (s *Service) DownloadDocument(documentID, userID, deviceID, ip string) ([]byte, string, error) {
	ok, err := s.audit.CheckDownloadPermission(userID, "candidate_document", documentID)
	if err != nil {
		return nil, "", err
	}
	if !ok {
		return nil, "", errors.New("download not permitted")
	}

	doc, err := s.getDocument(documentID)
	if err != nil {
		return nil, "", err
	}

	data, err := os.ReadFile(doc.FilePath)
	if err != nil {
		return nil, "", errors.New("file not found on disk")
	}

	// Verify file integrity before serving.
	computed := fmt.Sprintf("%x", sha256.Sum256(data))
	if computed != doc.SHA256Hash {
		return nil, "", errors.New("file integrity check failed: hash mismatch")
	}

	// Apply watermark transform if the document requires it.
	watermarked := false
	if doc.WatermarkEnabled {
		marked, applied := applyWatermark(data, doc.MimeType)
		if applied {
			data = marked
			watermarked = true
		}
		_ = s.audit.Log(userID, "DOCUMENT_WATERMARKED", "candidate_document", documentID, deviceID, ip,
			nil, map[string]interface{}{"fileName": doc.FileName, "applied": applied})
	}

	// Log the download event using the unified audit schema.
	_ = s.audit.Log(userID, "DOCUMENT_DOWNLOADED", "candidate_document", documentID, deviceID, ip,
		nil, map[string]interface{}{"fileName": doc.FileName, "watermarked": watermarked})

	return data, doc.FileName, nil
}

// applyWatermark applies a diagonal stripe watermark to JPEG or PNG images.
// Returns the transformed bytes and whether watermarking was applied.
func applyWatermark(data []byte, mimeType string) ([]byte, bool) {
	switch mimeType {
	case "image/jpeg", "image/jpg":
		result, err := watermarkImage(data, "jpeg")
		if err != nil {
			return data, false
		}
		return result, true
	case "image/png":
		result, err := watermarkImage(data, "png")
		if err != nil {
			return data, false
		}
		return result, true
	default:
		return data, false
	}
}

// watermarkImage decodes an image, draws a semi-transparent red diagonal stripe overlay,
// and re-encodes it in the original format.
func watermarkImage(data []byte, format string) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	dst := image.NewRGBA(bounds)
	draw.Draw(dst, bounds, img, bounds.Min, draw.Src)

	// Draw semi-transparent red diagonal stripes every 60 pixels.
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if (x+y)%60 < 12 {
				orig := dst.RGBAAt(x, y)
				dst.SetRGBA(x, y, blendRed(orig, 80))
			}
		}
	}

	var buf bytes.Buffer
	switch format {
	case "jpeg":
		if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 85}); err != nil {
			return nil, err
		}
	default:
		if err := png.Encode(&buf, dst); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

// blendRed blends a semi-transparent red over the given background pixel.
func blendRed(bg color.RGBA, alpha uint8) color.RGBA {
	a := uint32(alpha)
	inv := uint32(255 - alpha)
	return color.RGBA{
		R: uint8((uint32(bg.R)*inv + 255*a) / 255),
		G: uint8((uint32(bg.G)*inv) / 255),
		B: uint8((uint32(bg.B)*inv) / 255),
		A: 255,
	}
}

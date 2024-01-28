package zim

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"gitlab.com/wpetit/goweb/logger"
)

type readerTestCase struct {
	UUID       string `json:"uuid"`
	EntryCount uint32 `json:"entryCount"`
	Entries    []struct {
		Namespace   Namespace `json:"namespace"`
		URL         string    `json:"url"`
		Size        int64     `json:"size"`
		Compression int       `json:"compression"`
		MimeType    string    `json:"mimeType"`
		Title       string    `json:"title"`
	} `json:"entries"`
}

func TestReader(t *testing.T) {
	if testing.Verbose() {
		logger.SetLevel(logger.LevelDebug)
		logger.SetFormat(logger.FormatHuman)
	}

	files, err := filepath.Glob("testdata/*.zim")
	if err != nil {
		t.Fatalf("%+v", errors.WithStack(err))
	}

	for _, zf := range files {
		testName := filepath.Base(zf)
		testCase, err := loadZimFileTestCase(zf)
		if err != nil {
			t.Fatalf("%+v", errors.WithStack(err))
		}

		t.Run(testName, func(t *testing.T) {
			reader, err := Open(zf)
			if err != nil {
				t.Fatalf("%+v", errors.WithStack(err))
			}

			defer func() {
				if err := reader.Close(); err != nil {
					t.Errorf("%+v", errors.WithStack(err))
				}
			}()

			if e, g := testCase.UUID, reader.UUID(); e != g {
				t.Errorf("reader.UUID(): expected '%s', got '%s'", e, g)
			}

			if e, g := testCase.EntryCount, reader.EntryCount(); e != g {
				t.Errorf("reader.EntryCount(): expected '%v', got '%v'", e, g)
			}

			if testCase.Entries == nil {
				return
			}

			for _, entryTestCase := range testCase.Entries {
				testName := fmt.Sprintf("Entry/%s/%s", entryTestCase.Namespace, entryTestCase.URL)
				t.Run(testName, func(t *testing.T) {
					entry, err := reader.EntryWithURL(entryTestCase.Namespace, entryTestCase.URL)
					if err != nil {
						t.Fatalf("%+v", errors.WithStack(err))
					}

					content, err := entry.Redirect()
					if err != nil {
						t.Errorf("%+v", errors.WithStack(err))
					}

					if e, g := entryTestCase.MimeType, content.MimeType(); e != g {
						t.Errorf("content.MimeType(): expected '%v', got '%v'", e, g)
					}

					if e, g := entryTestCase.Title, content.Title(); e != g {
						t.Errorf("content.Title(): expected '%v', got '%v'", e, g)
					}

					compression, err := content.Compression()
					if err != nil {
						t.Errorf("%+v", errors.WithStack(err))
					}

					if e, g := entryTestCase.Compression, compression; e != g {
						t.Errorf("content.Compression(): expected '%v', got '%v'", e, g)
					}

					contentReader, err := content.Reader()
					if err != nil {
						t.Errorf("%+v", errors.WithStack(err))
					}

					size, err := contentReader.Size()
					if err != nil {
						t.Errorf("%+v", errors.WithStack(err))
					}

					if e, g := entryTestCase.Size, size; e != g {
						t.Errorf("content.Size(): expected '%v', got '%v'", e, g)
					}
				})
			}
		})
	}
}

func loadZimFileTestCase(zimFile string) (*readerTestCase, error) {
	testCaseFile, _ := strings.CutSuffix(zimFile, ".zim")

	data, err := os.ReadFile(testCaseFile + ".json")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	testCase := &readerTestCase{}
	if err := json.Unmarshal(data, testCase); err != nil {
		return nil, errors.WithStack(err)
	}

	return testCase, nil
}

package zim

import (
	"io"

	"github.com/pkg/errors"
)

type MetadataKey string

// See https://wiki.openzim.org/wiki/Metadata
const (
	MetadataName                 MetadataKey = "Name"
	MetadataTitle                MetadataKey = "Title"
	MetadataDescription          MetadataKey = "Description"
	MetadataLongDescription      MetadataKey = "LongDescription"
	MetadataCreator              MetadataKey = "Creator"
	MetadataTags                 MetadataKey = "Tags"
	MetadataDate                 MetadataKey = "Date"
	MetadataPublisher            MetadataKey = "Publisher"
	MetadataFlavour              MetadataKey = "Flavour"
	MetadataSource               MetadataKey = "Source"
	MetadataLanguage             MetadataKey = "Language"
	MetadataIllustration48x48at1 MetadataKey = "Illustration_48x48@1"
	MetadataIllustration96x96at2 MetadataKey = "Illustration_96x96@2"
)

var knownKeys = []MetadataKey{
	MetadataName,
	MetadataTitle,
	MetadataDescription,
	MetadataLongDescription,
	MetadataCreator,
	MetadataPublisher,
	MetadataLanguage,
	MetadataTags,
	MetadataDate,
	MetadataFlavour,
	MetadataSource,
	MetadataIllustration48x48at1,
	MetadataIllustration96x96at2,
}

// Metadata returns a copy of the internal metadata map of the ZIM file.
func (r *Reader) Metadata(keys ...MetadataKey) (map[MetadataKey]string, error) {
	if len(keys) == 0 {
		keys = knownKeys
	}

	metadata := make(map[MetadataKey]string)

	for _, key := range keys {
		entry, err := r.EntryWithURL(V5NamespaceMetadata, string(key))
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}

			return nil, errors.WithStack(err)
		}

		content, err := entry.Redirect()
		if err != nil {
			return nil, errors.WithStack(err)
		}

		reader, err := content.Reader()
		if err != nil {
			return nil, errors.WithStack(err)
		}

		data, err := io.ReadAll(reader)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		metadata[key] = string(data)
	}

	return metadata, nil
}

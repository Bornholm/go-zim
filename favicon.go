package zim

import "github.com/pkg/errors"

func (r *Reader) Favicon() (*ContentEntry, error) {
	illustration, err := r.getMetadataIllustration()
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, errors.WithStack(err)
	}

	if illustration != nil {
		return illustration, nil
	}

	namespaces := []Namespace{V5NamespaceLayout, V5NamespaceImageFile}
	urls := []string{"favicon", "favicon.png"}

	for _, ns := range namespaces {
		for _, url := range urls {
			entry, err := r.EntryWithURL(ns, url)
			if err != nil && !errors.Is(err, ErrNotFound) {
				return nil, errors.WithStack(err)
			}

			if errors.Is(err, ErrNotFound) {
				continue
			}

			content, err := entry.Redirect()
			if err != nil {
				return nil, errors.WithStack(err)
			}

			return content, nil
		}
	}

	return nil, errors.WithStack(ErrNotFound)
}

func (r *Reader) getMetadataIllustration() (*ContentEntry, error) {
	keys := []MetadataKey{MetadataIllustration96x96at2, MetadataIllustration48x48at1}

	metadata, err := r.Metadata(keys...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	for _, k := range keys {
		if _, exists := metadata[k]; exists {
			entry, err := r.EntryWithURL(V5NamespaceMetadata, string(k))
			if err != nil {
				return nil, errors.WithStack(err)
			}

			content, err := entry.Redirect()
			if err != nil {
				return nil, errors.WithStack(err)
			}

			return content, nil
		}
	}

	return nil, errors.WithStack(ErrNotFound)
}

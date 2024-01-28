package zim

import "time"

type Options struct {
	URLCacheSize int
	URLCacheTTL  time.Duration
	CacheSize    int
}

type OptionFunc func(opts *Options)

func NewOptions(funcs ...OptionFunc) *Options {
	funcs = append([]OptionFunc{
		WithCacheSize(2048),
	}, funcs...)

	opts := &Options{}
	for _, fn := range funcs {
		fn(opts)
	}

	return opts
}

func WithCacheSize(size int) OptionFunc {
	return func(opts *Options) {
		opts.CacheSize = size
	}
}

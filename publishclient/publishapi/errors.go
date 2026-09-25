package publishapi

import (
	"errors"

	"github.com/anyproto/any-sync/net/rpc/rpcerr"
	"storj.io/drpc/drpcerr"
)

var (
	// Preserve the deployed publisher's wire codes without registering them in
	// any-sync's process-global registry: pubsub also uses the 1100 range.
	ErrUnexpected   = drpcerr.WithCode(errors.New("unexpected error"), uint64(ErrCodes_ErrorOffset+ErrCodes_Unexpected))
	ErrNotFound     = drpcerr.WithCode(errors.New("not found"), uint64(ErrCodes_ErrorOffset+ErrCodes_NotFound))
	ErrAccessDenied = drpcerr.WithCode(errors.New("access denied"), uint64(ErrCodes_ErrorOffset+ErrCodes_AccessDenied))
	ErrUriNotUnique = drpcerr.WithCode(errors.New("uri already taken"), uint64(ErrCodes_ErrorOffset+ErrCodes_UriNotUnique))
)

// UnwrapError decodes errors returned by WebPublisher RPCs. Publisher codes
// must be resolved in this service's context, since other protocols can use
// the same codes. Shared transport errors still use any-sync's registry.
func UnwrapError(err error) error {
	code := drpcerr.Code(err)
	switch code {
	case uint64(ErrCodes_ErrorOffset + ErrCodes_Unexpected):
		return ErrUnexpected
	case uint64(ErrCodes_ErrorOffset + ErrCodes_NotFound):
		return ErrNotFound
	case uint64(ErrCodes_ErrorOffset + ErrCodes_AccessDenied):
		return ErrAccessDenied
	case uint64(ErrCodes_ErrorOffset + ErrCodes_UriNotUnique):
		return ErrUriNotUnique
	}
	if code >= uint64(ErrCodes_ErrorOffset) && code < uint64(ErrCodes_ErrorOffset)+100 {
		// An unknown publisher code must not become an unrelated pubsub error.
		return err
	}
	return rpcerr.Unwrap(err)
}

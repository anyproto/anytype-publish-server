package publishapi

import (
	"errors"

	"github.com/anyproto/any-sync/net/rpc/rpcerr"
)

var (
	errGroup = rpcerr.ErrGroup(ErrCodes_ErrorOffset)

	ErrUnexpected   = errGroup.Register(errors.New("unexpected error"), uint64(ErrCodes_Unexpected))
	ErrNotFound     = errGroup.Register(errors.New("not found"), uint64(ErrCodes_NotFound))
	ErrAccessDenied = errGroup.Register(errors.New("access denied"), uint64(ErrCodes_AccessDenied))
	ErrUriNotUnique = errGroup.Register(errors.New("uri already taken"), uint64(ErrCodes_UriNotUnique))
)

package publishapi_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/anyproto/any-sync/net/rpc/rpcerr"
	"storj.io/drpc/drpcerr"

	"github.com/anyproto/anytype-publish-server/publishclient/publishapi"
)

// Model another imported service owning the same global codes. Importing
// publishapi must neither panic nor require ownership of those registry slots.
var otherServiceErrors = map[uint64]error{
	1100: rpcerr.RegisterErr(errors.New("other unexpected"), 1100),
	1101: rpcerr.RegisterErr(errors.New("other not found"), 1101),
	1102: rpcerr.RegisterErr(errors.New("other access denied"), 1102),
	1103: rpcerr.RegisterErr(errors.New("other uri not unique"), 1103),
	1104: rpcerr.RegisterErr(errors.New("other service only"), 1104),
}

func TestPublisherWireErrors(t *testing.T) {
	for _, tc := range []struct {
		code uint64
		err  error
	}{
		{1100, publishapi.ErrUnexpected},
		{1101, publishapi.ErrNotFound},
		{1102, publishapi.ErrAccessDenied},
		{1103, publishapi.ErrUriNotUnique},
	} {
		t.Run(tc.err.Error(), func(t *testing.T) {
			// A server using these sentinels still sends its original wire code.
			if got := drpcerr.Code(tc.err); got != tc.code {
				t.Fatalf("wire code = %d, want %d", got, tc.code)
			}
			wireErr := drpcerr.WithCode(errors.New("remote error"), tc.code)
			got := publishapi.UnwrapError(fmt.Errorf("request failed: %w", wireErr))
			if !errors.Is(got, tc.err) {
				t.Fatalf("decoded %v, want %v", got, tc.err)
			}
			if rpcerr.Err(tc.code) != otherServiceErrors[tc.code] {
				t.Fatal("publisher modified another service's global registration")
			}
		})
	}
}

func TestUnwrapErrorFallbacks(t *testing.T) {
	plain := errors.New("connection failed")
	unknownPublisher := drpcerr.WithCode(errors.New("new publisher error"), 1104)
	unknownShared := drpcerr.WithCode(errors.New("unknown transport error"), 9999)
	for _, tc := range []struct {
		name string
		in   error
		want error
	}{
		{"nil", nil, nil},
		{"uncoded", plain, plain},
		{"unknown publisher", unknownPublisher, unknownPublisher},
		{"shared transport", drpcerr.WithCode(errors.New("remote closed"), 2), rpcerr.Closed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := publishapi.UnwrapError(tc.in); got != tc.want {
				t.Fatalf("decoded %v, want %v", got, tc.want)
			}
		})
	}
	if got := publishapi.UnwrapError(unknownShared); drpcerr.Code(got) != 9999 {
		t.Fatalf("unknown shared error lost its wire code: %v", got)
	}
}

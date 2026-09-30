package graphann

import (
	"sort"
	"testing"

	"github.com/graphann/graphann-client-go/generated"
)

func TestErrorCodesMatchSpecEnum(t *testing.T) {
	spec := []string{
		string(generated.APIErrorCodeBadGateway), string(generated.APIErrorCodeBadRequest),
		string(generated.APIErrorCodeConflict), string(generated.APIErrorCodeForbidden),
		string(generated.APIErrorCodeIndexBuilding), string(generated.APIErrorCodeIndexNotReady),
		string(generated.APIErrorCodeInsufficientStorage), string(generated.APIErrorCodeInternalError),
		string(generated.APIErrorCodeNotFound), string(generated.APIErrorCodeNotImplemented),
		string(generated.APIErrorCodePayloadTooLarge), string(generated.APIErrorCodeQuotaExceeded),
		string(generated.APIErrorCodeRateLimited), string(generated.APIErrorCodeServiceUnavailable),
		string(generated.APIErrorCodeUnauthorized), string(generated.APIErrorCodeUnprocessableEntity),
		string(generated.APIErrorCodeValidationError),
	}
	sdk := []string{
		ErrorCodeInternal, ErrorCodeBadRequest, ErrorCodeUnauthorized, ErrorCodeForbidden,
		ErrorCodeNotFound, ErrorCodeConflict, ErrorCodeQuotaExceeded, ErrorCodeRateLimited,
		ErrorCodeValidation, ErrorCodeIndexNotReady, ErrorCodeIndexBuilding,
		ErrorCodeServiceUnavailable, ErrorCodePayloadTooLarge, ErrorCodeInsufficientStorage,
		ErrorCodeNotImplemented, ErrorCodeBadGateway, ErrorCodeUnprocessable,
	}
	sort.Strings(spec)
	sort.Strings(sdk)
	if len(spec) != len(sdk) {
		t.Fatalf("SDK has %d codes, spec enum has %d", len(sdk), len(spec))
	}
	for i := range spec {
		if spec[i] != sdk[i] {
			t.Errorf("code mismatch at %d: sdk %q, spec %q", i, sdk[i], spec[i])
		}
		if !generated.APIErrorCode(sdk[i]).Valid() {
			t.Errorf("%q is not a valid spec code", sdk[i])
		}
	}
}

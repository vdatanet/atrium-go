package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestTheNoContentShapeRemovesAContentTypeAnEarlierStageLeftBehind is the one
// line of writeNoContent, asserted where it can be.
//
// No route reaches writeNoContent with a content type already in the header
// map, so through the handler the Del changes no byte and a mutation removing
// it would survive the whole suite — the position WriteControllerRefusal's
// declared length is in, and recorded there rather than tested. This one can be
// tested, because the shape is a function: hand it a header map with a content
// type in it, which is exactly the state a buffering stage or a handler that
// changed its mind would produce, and the absence becomes a property of this
// code rather than of what happened to run before it.
//
// It matters because net/http will not remove it: a Content-Type set before a
// `204` reaches the wire, where a Content-Length does not
// [measurement: net/http, Go 1.27.0, 2026-09-03].
func TestTheNoContentShapeRemovesAContentTypeAnEarlierStageLeftBehind(t *testing.T) {
	recorder := httptest.NewRecorder()
	recorder.Header().Set("Content-Type", "application/json; charset=utf-8")

	writeNoContent(recorder)

	if recorder.Code != http.StatusNoContent {
		t.Errorf("the status is %d, want 204", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); got != "" {
		t.Errorf("the content type is %q, and spec 3.6's 204 declares none", got)
	}
	if body := recorder.Body.String(); body != "" {
		t.Errorf("the body is %q, and this shape has none", body)
	}
}

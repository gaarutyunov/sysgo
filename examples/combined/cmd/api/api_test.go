package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/gaarutyunov/sysgo/examples/combined/api"
	"github.com/gaarutyunov/sysgo/examples/combined/internal/order/app/usecase"
)

func newRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api.RegisterHandlers(r, NewServer(usecase.NewPlaceOrderInteractor()))
	return r
}

func post(t *testing.T, body api.OrderContextOrder) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newRouter().ServeHTTP(w, req)
	return w
}

// TestPlaceOrderAPI drives POST /orders through the generated server into the
// DDD PlaceOrder use case and back.
func TestPlaceOrderAPI(t *testing.T) {
	w := post(t, api.OrderContextOrder{Id: "o-1"})
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	var got api.OrderContextOrder
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Id != "o-1" {
		t.Errorf("echoed id = %q, want o-1", got.Id)
	}
}

// TestPlaceOrderAPIRejectsEmptyID shows the use case's rule surfacing over REST.
func TestPlaceOrderAPIRejectsEmptyID(t *testing.T) {
	if w := post(t, api.OrderContextOrder{Id: ""}); w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

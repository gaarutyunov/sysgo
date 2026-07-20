package combined_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.temporal.io/sdk/testsuite"

	"github.com/gaarutyunov/sysgo/examples/combined/api"
	adapterhttp "github.com/gaarutyunov/sysgo/examples/combined/internal/order/adapter/in/http"
	adaptertemporal "github.com/gaarutyunov/sysgo/examples/combined/internal/order/adapter/in/temporal"
	"github.com/gaarutyunov/sysgo/examples/combined/internal/order/adapter/out/repository"
	"github.com/gaarutyunov/sysgo/examples/combined/internal/order/app/usecase"
	"github.com/gaarutyunov/sysgo/examples/combined/orders"
)

// TestSameUseCaseFromBothTransports is the point of the combined example: a
// single PlaceOrder use case (over a single repository) is driven from BOTH the
// REST driving adapter and the Temporal driving adapter. The same business logic
// runs whichever entrypoint the caller used.
func TestSameUseCaseFromBothTransports(t *testing.T) {
	repo := repository.NewOrderRepositoryImpl()
	placeOrder := usecase.NewPlaceOrderInteractor(repo)

	// (1) Drive the use case through the REST driving adapter.
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api.RegisterHandlers(router, adapterhttp.NewPlaceOrderUseCaseHandler(placeOrder))

	body, _ := json.Marshal(api.OrderContextOrder{Id: "rest-1"})
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("REST status = %d, want 201; body=%s", w.Code, w.Body.String())
	}

	// (2) Drive the SAME use case through the Temporal driving adapter.
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	env.RegisterActivity(adaptertemporal.NewActivities(placeOrder))
	env.ExecuteWorkflow(orders.ProcessOrderWorkflow, orders.Order{Id: "temporal-1"})
	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow error: %v", err)
	}

	// Both transports drove the same use case into the same repository.
	if _, ok := repo.Get("rest-1"); !ok {
		t.Error("order placed over REST was not persisted")
	}
	if _, ok := repo.Get("temporal-1"); !ok {
		t.Error("order placed over Temporal was not persisted")
	}
	if got := repo.Len(); got != 2 {
		t.Errorf("repository holds %d orders, want 2", got)
	}
}

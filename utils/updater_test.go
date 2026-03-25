// updater_test.go contains tests for the Updater type and its Apply method.
// Tests here should cover Apply behavior using a mock k8s client — do not
// add tests that require a real cluster or envtest environment.
package utils

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	core "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// mockClient is a partial mock of client.Client for unit testing. Only the methods
// needed by the test are overridden; all others panic via the embedded interface.
// The embedded client.Client satisfies the interface at compile time without requiring
// every method to be implemented — any unoverridden method will panic if called at runtime.
type mockClient struct {
	client.Client
	updateFunc func(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error
}

// Update delegates to updateFunc, allowing tests to inject custom update behavior.
func (m *mockClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return m.updateFunc(ctx, obj, opts...)
}

func TestApplyRetryOnConflict(t *testing.T) {
	calls := 0
	mock := &mockClient{
		updateFunc: func(_ context.Context, _ client.Object, _ ...client.UpdateOption) error {
			calls++
			if calls == 1 {
				return k8serr.NewConflict(
					schema.GroupResource{Group: "", Resource: "configmaps"},
					"test-resource",
					fmt.Errorf("the object has been modified"),
				)
			}
			return nil
		},
	}

	u := Updater(true)
	obj := &core.ConfigMap{}
	obj.SetName("test-resource")
	obj.SetNamespace("default")

	err := u.Apply(context.Background(), mock, obj)
	assert.NoError(t, err)
	assert.Equal(t, 2, calls, "should retry update once after conflict")
}

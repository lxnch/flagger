package canary

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	flaggerv1 "github.com/weaveworks/flagger/pkg/apis/flagger/v1beta1"
)

func TestDaemonSetController_IsReady(t *testing.T) {
	mocks := newDaemonSetFixture()
	err := mocks.controller.Initialize(mocks.canary, true)
	assert.NoError(t, err, "Expected primary readiness check to fail")

	err = mocks.controller.IsPrimaryReady(mocks.canary)
	require.NoError(t, err)

	_, err = mocks.controller.IsCanaryReady(mocks.canary)
	require.NoError(t, err)
}

func TestDaemonSetController_isDaemonSetReady(t *testing.T) {
	mocks := newDaemonSetFixture()
	cd := &flaggerv1.Canary{}

	// observed generation is less than desired generation
	ds := &appsv1.DaemonSet{Status: appsv1.DaemonSetStatus{}}
	ds.Status.ObservedGeneration--
	retyable, err := mocks.controller.isDaemonSetReady(cd, ds)
	require.Error(t, err)
	require.True(t, retyable)

	// succeeded
	ds = &appsv1.DaemonSet{Status: appsv1.DaemonSetStatus{
		UpdatedNumberScheduled: 1,
		DesiredNumberScheduled: 1,
		NumberAvailable:        1,
	}}
	retyable, err = mocks.controller.isDaemonSetReady(cd, ds)
	require.NoError(t, err)
	require.True(t, retyable)

	// deadline exceeded
	ds = &appsv1.DaemonSet{Status: appsv1.DaemonSetStatus{
		UpdatedNumberScheduled: 0,
		DesiredNumberScheduled: 1,
	}}
	cd.Status.LastTransitionTime = metav1.Now()
	cd.Spec.ProgressDeadlineSeconds = int32p(-1e6)
	retyable, err = mocks.controller.isDaemonSetReady(cd, ds)
	require.Error(t, err)
	require.False(t, retyable)

	// only newCond not satisfied
	ds = &appsv1.DaemonSet{Status: appsv1.DaemonSetStatus{
		UpdatedNumberScheduled: 0,
		DesiredNumberScheduled: 1,
		NumberAvailable:        1,
	}}
	cd.Spec.ProgressDeadlineSeconds = int32p(1e6)
	retyable, err = mocks.controller.isDaemonSetReady(cd, ds)
	require.Error(t, err)
	require.True(t, retyable)
	require.True(t, strings.Contains(err.Error(), "new pods"))

	// only availableCond not satisfied
	ds = &appsv1.DaemonSet{Status: appsv1.DaemonSetStatus{
		UpdatedNumberScheduled: 1,
		DesiredNumberScheduled: 1,
		NumberAvailable:        0,
	}}
	retyable, err = mocks.controller.isDaemonSetReady(cd, ds)
	require.Error(t, err)
	require.True(t, retyable)
	require.True(t, strings.Contains(err.Error(), "available"))
}

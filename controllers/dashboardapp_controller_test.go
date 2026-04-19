package controllers

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	dashboardv1alpha1 "github.com/fredericrous/duro-operator/api/v1alpha1"
)

var _ = Describe("DashboardApp controller", func() {
	const (
		timeout  = 10 * time.Second
		interval = 250 * time.Millisecond
	)

	newApp := func(name string) *dashboardv1alpha1.DashboardApp {
		return &dashboardv1alpha1.DashboardApp{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "default",
			},
			Spec: dashboardv1alpha1.DashboardAppSpec{
				Name:     name,
				URL:      "https://example.test/" + name,
				Category: "media",
				Icon:     "<svg/>",
				Groups:   []string{"users"},
				Priority: 100,
			},
		}
	}

	Context("status.observedGeneration", func() {
		It("is set to metadata.generation after the first reconcile", func() {
			app := newApp("obs-gen-initial")
			Expect(k8sClient.Create(ctx, app)).To(Succeed())

			key := types.NamespacedName{Name: app.Name, Namespace: app.Namespace}

			Eventually(func(g Gomega) {
				var got dashboardv1alpha1.DashboardApp
				g.Expect(k8sClient.Get(ctx, key, &got)).To(Succeed())
				g.Expect(got.Status.Ready).To(BeTrue())
				g.Expect(got.Status.ObservedGeneration).To(Equal(got.Generation))
				g.Expect(got.Status.LastSyncedAt).NotTo(BeNil())
			}, timeout, interval).Should(Succeed())
		})

		It("tracks generation across spec updates", func() {
			app := newApp("obs-gen-update")
			Expect(k8sClient.Create(ctx, app)).To(Succeed())

			key := types.NamespacedName{Name: app.Name, Namespace: app.Namespace}

			// Wait for initial convergence.
			Eventually(func(g Gomega) {
				var got dashboardv1alpha1.DashboardApp
				g.Expect(k8sClient.Get(ctx, key, &got)).To(Succeed())
				g.Expect(got.Status.ObservedGeneration).To(Equal(got.Generation))
			}, timeout, interval).Should(Succeed())

			// Update the spec — bumps Generation.
			var current dashboardv1alpha1.DashboardApp
			Expect(k8sClient.Get(ctx, key, &current)).To(Succeed())
			originalGen := current.Generation
			current.Spec.URL = "https://example.test/updated"
			Expect(k8sClient.Update(ctx, &current)).To(Succeed())

			Eventually(func(g Gomega) {
				var got dashboardv1alpha1.DashboardApp
				g.Expect(k8sClient.Get(ctx, key, &got)).To(Succeed())
				g.Expect(got.Generation).To(BeNumerically(">", originalGen))
				g.Expect(got.Status.ObservedGeneration).To(Equal(got.Generation))
			}, timeout, interval).Should(Succeed())
		})
	})

	Context("status-write quiescence", func() {
		It("stops rewriting LastSyncedAt once the spec is fully observed", func() {
			// The bug: controllers wrote Status.LastSyncedAt = now on every
			// reconcile, cascading watch events into a tight loop. Fix: skip
			// the write when Ready && ObservedGeneration == Generation.
			//
			// Sample LastSyncedAt twice, 2s apart. If the controller is still
			// looping, the timestamp will keep moving; if it's quiescent,
			// it won't.
			app := newApp("quiet-after-convergence")
			Expect(k8sClient.Create(ctx, app)).To(Succeed())

			key := types.NamespacedName{Name: app.Name, Namespace: app.Namespace}

			var first *metav1.Time
			Eventually(func(g Gomega) {
				var got dashboardv1alpha1.DashboardApp
				g.Expect(k8sClient.Get(ctx, key, &got)).To(Succeed())
				g.Expect(got.Status.ObservedGeneration).To(Equal(got.Generation))
				first = got.Status.LastSyncedAt
				g.Expect(first).NotTo(BeNil())
			}, timeout, interval).Should(Succeed())

			time.Sleep(2 * time.Second)

			var got dashboardv1alpha1.DashboardApp
			Expect(k8sClient.Get(ctx, key, &got)).To(Succeed())
			Expect(got.Status.LastSyncedAt.Equal(first)).To(BeTrue(),
				"LastSyncedAt should not advance while spec is unchanged (got %v, was %v)",
				got.Status.LastSyncedAt, first)
		})
	})
})

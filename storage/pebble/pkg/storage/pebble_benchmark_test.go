package storage

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8storage "k8s.io/apiserver/pkg/storage"

	k1sstorage "github.com/dtomasi/k1s/core/pkg/storage"
)

// BenchmarkPebbleStorage_Create measures create operation performance
func BenchmarkPebbleStorage_Create(b *testing.B) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "pebble-benchmark-create-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	config := k1sstorage.Config{}

	storage := NewPebbleStorageWithPath(tempDir, config)
	defer storage.Close()

	testObject := &TestObject{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "test/v1",
			Kind:       "TestObject",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "benchmark-object",
			Namespace: "default",
		},
		Spec: TestSpec{
			Name:        "Benchmark Object",
			Description: "A test object for benchmarking",
		},
		Status: TestStatus{
			Phase: "Active",
		},
	}

	b.ResetTimer()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("benchmark-objects/create-test-%d", i)
		obj := testObject.DeepCopyObject().(*TestObject)
		obj.Name = fmt.Sprintf("benchmark-object-%d", i)

		err := storage.Create(ctx, key, obj, nil, 0)
		if err != nil {
			b.Fatalf("Create failed: %v", err)
		}
	}

	b.StopTimer()
}

// BenchmarkPebbleStorage_Get measures get operation performance
func BenchmarkPebbleStorage_Get(b *testing.B) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "pebble-benchmark-get-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	config := k1sstorage.Config{}

	storage := NewPebbleStorageWithPath(tempDir, config)
	defer storage.Close()

	testObject := &TestObject{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "test/v1",
			Kind:       "TestObject",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "benchmark-object",
			Namespace: "default",
		},
		Spec: TestSpec{
			Name:        "Benchmark Object",
			Description: "A test object for benchmarking",
		},
		Status: TestStatus{
			Phase: "Active",
		},
	}

	// Pre-populate with test data
	numObjects := 1000
	for i := 0; i < numObjects; i++ {
		key := fmt.Sprintf("benchmark-objects/get-test-%d", i)
		obj := testObject.DeepCopyObject().(*TestObject)
		obj.Name = fmt.Sprintf("benchmark-object-%d", i)

		err := storage.Create(ctx, key, obj, nil, 0)
		if err != nil {
			b.Fatalf("Setup failed: %v", err)
		}
	}

	b.ResetTimer()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("benchmark-objects/get-test-%d", i%numObjects)
		retrieved := &TestObject{}

		err := storage.Get(ctx, key, k8storage.GetOptions{}, retrieved)
		if err != nil {
			b.Fatalf("Get failed: %v", err)
		}
	}

	b.StopTimer()
}

// BenchmarkPebbleStorage_Delete measures delete operation performance
func BenchmarkPebbleStorage_Delete(b *testing.B) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "pebble-benchmark-delete-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	config := k1sstorage.Config{}

	storage := NewPebbleStorageWithPath(tempDir, config)
	defer storage.Close()

	testObject := &TestObject{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "test/v1",
			Kind:       "TestObject",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "benchmark-object",
			Namespace: "default",
		},
		Spec: TestSpec{
			Name:        "Benchmark Object",
			Description: "A test object for benchmarking",
		},
		Status: TestStatus{
			Phase: "Active",
		},
	}

	// Pre-populate with test data
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("benchmark-objects/delete-test-%d", i)
		obj := testObject.DeepCopyObject().(*TestObject)
		obj.Name = fmt.Sprintf("benchmark-object-%d", i)

		err := storage.Create(ctx, key, obj, nil, 0)
		if err != nil {
			b.Fatalf("Setup failed: %v", err)
		}
	}

	b.ResetTimer()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("benchmark-objects/delete-test-%d", i)
		out := &TestObject{}

		err := storage.Delete(ctx, key, out, nil, nil, nil)
		if err != nil {
			b.Fatalf("Delete failed: %v", err)
		}
	}

	b.StopTimer()
}

// BenchmarkPebbleStorage_List measures list operation performance
func BenchmarkPebbleStorage_List(b *testing.B) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "pebble-benchmark-list-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	config := k1sstorage.Config{}

	storage := NewPebbleStorageWithPath(tempDir, config)
	defer storage.Close()

	testObject := &TestObject{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "test/v1",
			Kind:       "TestObject",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "benchmark-object",
			Namespace: "default",
		},
		Spec: TestSpec{
			Name:        "Benchmark Object",
			Description: "A test object for benchmarking",
		},
		Status: TestStatus{
			Phase: "Active",
		},
	}

	// Pre-populate with test data
	numObjects := 100
	for i := 0; i < numObjects; i++ {
		key := fmt.Sprintf("benchmark-objects/list-test-%d", i)
		obj := testObject.DeepCopyObject().(*TestObject)
		obj.Name = fmt.Sprintf("benchmark-object-%d", i)

		err := storage.Create(ctx, key, obj, nil, 0)
		if err != nil {
			b.Fatalf("Setup failed: %v", err)
		}
	}

	testList := &TestObjectList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "test/v1",
			Kind:       "TestObjectList",
		},
	}

	b.ResetTimer()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		err := storage.List(ctx, "benchmark-objects", k8storage.ListOptions{Recursive: true}, testList)
		if err != nil {
			b.Fatalf("List failed: %v", err)
		}
	}

	b.StopTimer()
}

// BenchmarkPebbleStorage_ConcurrentOperations measures concurrent operation performance
func BenchmarkPebbleStorage_ConcurrentOperations(b *testing.B) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "pebble-benchmark-concurrent-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	config := k1sstorage.Config{}

	storage := NewPebbleStorageWithPath(tempDir, config)
	defer storage.Close()

	testObject := &TestObject{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "test/v1",
			Kind:       "TestObject",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "benchmark-object",
			Namespace: "default",
		},
		Spec: TestSpec{
			Name:        "Benchmark Object",
			Description: "A test object for benchmarking",
		},
		Status: TestStatus{
			Phase: "Active",
		},
	}

	b.ResetTimer()
	b.StartTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("benchmark-objects/concurrent-test-%d", i)
			obj := testObject.DeepCopyObject().(*TestObject)
			obj.Name = fmt.Sprintf("benchmark-object-%d", i)

			err := storage.Create(ctx, key, obj, nil, 0)
			if err != nil {
				b.Fatalf("Concurrent create failed: %v", err)
			}
			i++
		}
	})

	b.StopTimer()
}

// BenchmarkPebbleStorage_MixedOperations measures mixed operation performance (realistic workload)
func BenchmarkPebbleStorage_MixedOperations(b *testing.B) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "pebble-benchmark-mixed-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	config := k1sstorage.Config{}

	storage := NewPebbleStorageWithPath(tempDir, config)
	defer storage.Close()

	testObject := &TestObject{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "test/v1",
			Kind:       "TestObject",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "benchmark-object",
			Namespace: "default",
		},
		Spec: TestSpec{
			Name:        "Benchmark Object",
			Description: "A test object for benchmarking",
		},
		Status: TestStatus{
			Phase: "Active",
		},
	}

	// Pre-populate with some test data
	numObjects := 500
	for i := 0; i < numObjects; i++ {
		key := fmt.Sprintf("benchmark-objects/mixed-test-%d", i)
		obj := testObject.DeepCopyObject().(*TestObject)
		obj.Name = fmt.Sprintf("benchmark-object-%d", i)

		err := storage.Create(ctx, key, obj, nil, 0)
		if err != nil {
			b.Fatalf("Setup failed: %v", err)
		}
	}

	b.ResetTimer()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		switch i % 4 {
		case 0: // Create (25%)
			key := fmt.Sprintf("benchmark-objects/mixed-create-%d", i)
			obj := testObject.DeepCopyObject().(*TestObject)
			obj.Name = fmt.Sprintf("benchmark-create-%d", i)

			err := storage.Create(ctx, key, obj, nil, 0)
			if err != nil {
				b.Fatalf("Mixed create failed: %v", err)
			}

		case 1, 2: // Get (50%)
			key := fmt.Sprintf("benchmark-objects/mixed-test-%d", i%numObjects)
			retrieved := &TestObject{}

			err := storage.Get(ctx, key, k8storage.GetOptions{}, retrieved)
			if err != nil {
				b.Fatalf("Mixed get failed: %v", err)
			}

		case 3: // Delete (25%)
			if i < numObjects {
				key := fmt.Sprintf("benchmark-objects/mixed-test-%d", i)
				out := &TestObject{}

				err := storage.Delete(ctx, key, out, nil, nil, nil)
				if err != nil {
					b.Fatalf("Mixed delete failed: %v", err)
				}
			}
		}
	}

	b.StopTimer()
}

// BenchmarkPebbleStorage_HighThroughput measures high-throughput performance (target: >3000 ops/sec)
func BenchmarkPebbleStorage_HighThroughput(b *testing.B) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "pebble-benchmark-throughput-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	config := k1sstorage.Config{}

	storage := NewPebbleStorageWithPath(tempDir, config)
	defer storage.Close()

	testObject := &TestObject{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "test/v1",
			Kind:       "TestObject",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "benchmark-object",
			Namespace: "default",
		},
		Spec: TestSpec{
			Name:        "Benchmark Object",
			Description: "A test object for benchmarking",
		},
		Status: TestStatus{
			Phase: "Active",
		},
	}

	// Measure operations per second
	const targetOpsPerSec = 3000
	const testDurationSec = 5

	b.ResetTimer()
	start := time.Now()
	operations := 0

	// Run for fixed duration to measure throughput
	for time.Since(start) < testDurationSec*time.Second {
		key := fmt.Sprintf("benchmark-objects/throughput-test-%d", operations)
		obj := testObject.DeepCopyObject().(*TestObject)
		obj.Name = fmt.Sprintf("benchmark-object-%d", operations)

		err := storage.Create(ctx, key, obj, nil, 0)
		if err != nil {
			b.Fatalf("Throughput test failed: %v", err)
		}
		operations++
	}

	duration := time.Since(start)
	opsPerSec := float64(operations) / duration.Seconds()

	b.ReportMetric(opsPerSec, "ops/sec")
	b.ReportMetric(float64(operations), "total_ops")
	b.ReportMetric(duration.Seconds(), "duration_sec")

	if opsPerSec < targetOpsPerSec {
		b.Logf("WARNING: Throughput %0.2f ops/sec is below target %d ops/sec", opsPerSec, targetOpsPerSec)
	} else {
		b.Logf("SUCCESS: Throughput %0.2f ops/sec exceeds target %d ops/sec", opsPerSec, targetOpsPerSec)
	}

	b.StopTimer()
}

// BenchmarkPebbleStorage_LowLatency measures low-latency performance (target: <10ms per operation)
func BenchmarkPebbleStorage_LowLatency(b *testing.B) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "pebble-benchmark-latency-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	config := k1sstorage.Config{}

	storage := NewPebbleStorageWithPath(tempDir, config)
	defer storage.Close()

	testObject := &TestObject{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "test/v1",
			Kind:       "TestObject",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "benchmark-object",
			Namespace: "default",
		},
		Spec: TestSpec{
			Name:        "Benchmark Object",
			Description: "A test object for benchmarking",
		},
		Status: TestStatus{
			Phase: "Active",
		},
	}

	const targetLatencyMs = 10
	var totalLatency time.Duration
	operations := 0

	b.ResetTimer()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("benchmark-objects/latency-test-%d", i)
		obj := testObject.DeepCopyObject().(*TestObject)
		obj.Name = fmt.Sprintf("benchmark-object-%d", i)

		start := time.Now()
		err := storage.Create(ctx, key, obj, nil, 0)
		latency := time.Since(start)

		if err != nil {
			b.Fatalf("Latency test failed: %v", err)
		}

		totalLatency += latency
		operations++
	}

	avgLatency := totalLatency / time.Duration(operations)
	avgLatencyMs := float64(avgLatency.Nanoseconds()) / 1_000_000

	b.ReportMetric(avgLatencyMs, "avg_latency_ms")
	b.ReportMetric(float64(operations), "total_ops")

	if avgLatencyMs > targetLatencyMs {
		b.Logf("WARNING: Average latency %.2fms exceeds target %dms", avgLatencyMs, targetLatencyMs)
	} else {
		b.Logf("SUCCESS: Average latency %.2fms is below target %dms", avgLatencyMs, targetLatencyMs)
	}

	b.StopTimer()
}

// BenchmarkPebbleStorage_CompactedOperations measures performance after compaction
func BenchmarkPebbleStorage_CompactedOperations(b *testing.B) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "pebble-benchmark-compact-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	config := k1sstorage.Config{}

	storage := NewPebbleStorageWithPath(tempDir, config)
	defer storage.Close()

	testObject := &TestObject{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "test/v1",
			Kind:       "TestObject",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "benchmark-object",
			Namespace: "default",
		},
		Spec: TestSpec{
			Name:        "Benchmark Object",
			Description: "A test object for benchmarking",
		},
		Status: TestStatus{
			Phase: "Active",
		},
	}

	// Pre-populate with lots of data to create compaction pressure
	numObjects := 5000
	for i := 0; i < numObjects; i++ {
		key := fmt.Sprintf("benchmark-objects/compact-test-%d", i)
		obj := testObject.DeepCopyObject().(*TestObject)
		obj.Name = fmt.Sprintf("benchmark-object-%d", i)

		err := storage.Create(ctx, key, obj, nil, 0)
		if err != nil {
			b.Fatalf("Setup failed: %v", err)
		}
	}

	// Force compaction
	err = storage.Compact(ctx)
	if err != nil {
		b.Fatalf("Compaction failed: %v", err)
	}

	b.ResetTimer()
	b.StartTimer()

	// Test performance after compaction
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("benchmark-objects/compact-test-%d", i%numObjects)
		retrieved := &TestObject{}

		err := storage.Get(ctx, key, k8storage.GetOptions{}, retrieved)
		if err != nil {
			b.Fatalf("Post-compaction get failed: %v", err)
		}
	}

	b.StopTimer()
}

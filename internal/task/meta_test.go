// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package task

import (
	"testing"
)

func TestRegisterTaskMetaDeduplication(t *testing.T) {
	// Clear dispatchableTasks for testing
	dispatchableTasksMutex.Lock()
	oldTasks := dispatchableTasks
	dispatchableTasks = nil
	dispatchableTasksMutex.Unlock()

	defer func() {
		dispatchableTasksMutex.Lock()
		dispatchableTasks = oldTasks
		dispatchableTasksMutex.Unlock()
	}()

	meta1 := TaskMeta{
		Type: "test_task",
		Name: "Test Task V1",
	}
	meta2 := TaskMeta{
		Type: "test_task",
		Name: "Test Task V2", // same Type, different Name
	}
	meta3 := TaskMeta{
		Type: "other_task",
		Name: "Other Task",
	}

	RegisterTaskMeta(meta1)
	RegisterTaskMeta(meta2) // Should be ignored since meta1 is already registered
	RegisterTaskMeta(meta3) // New type

	tasks := GetDispatchableTasks()
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}

	if tasks[0].Type != "test_task" || tasks[0].Name != "Test Task V1" {
		t.Errorf("expected test_task to preserve Test Task V1, got %+v", tasks[0])
	}

	if tasks[1].Type != "other_task" {
		t.Errorf("expected second task to be other_task, got %+v", tasks[1])
	}
}

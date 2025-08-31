package application

import (
"encoding/json"
"fmt"
"os"
"rsshub/internal/domain"
"syscall"
)

// Путь к файлу состояния
const StatePath = "/tmp/rsshub_state.json"

// IPCManager реализует интерфейс domain.IPCManager
type IPCManager struct{}

// NewIPCManager создает новый экземпляр IPCManager
func NewIPCManager() *IPCManager {
return &IPCManager{}
}

// SaveState сохраняет состояние агрегатора в файл
func (m *IPCManager) SaveState(state *domain.AggregatorState) error {
data, err := json.Marshal(state)
if err != nil {
return err
}

return os.WriteFile(StatePath, data, 0644)
}

// LoadState загружает состояние агрегатора из файла
func (m *IPCManager) LoadState() (*domain.AggregatorState, error) {
data, err := os.ReadFile(StatePath)
if err != nil {
if os.IsNotExist(err) {
return &domain.AggregatorState{Running: false}, nil
}
return nil, err
}

var state domain.AggregatorState
if err := json.Unmarshal(data, &state); err != nil {
return nil, err
}

// Проверяем, существует ли процесс
if state.Running && !m.IsProcessRunning(state.PID) {
return &domain.AggregatorState{Running: false}, nil
}

return &state, nil
}

// SignalProcess отправляет сигнал процессу
func (m *IPCManager) SignalProcess(state *domain.AggregatorState, signal int) error {
if !state.Running {
return fmt.Errorf("фоновый процесс не запущен")
}

proc, err := os.FindProcess(state.PID)
if err != nil {
return err
}

return proc.Signal(syscall.Signal(signal))
}

// IsProcessRunning проверяет, запущен ли процесс
func (m *IPCManager) IsProcessRunning(pid int) bool {
proc, err := os.FindProcess(pid)
if err != nil {
return false
}

err = proc.Signal(syscall.Signal(0))
return err == nil
}

package tools

import "log"

// RegisterAllProbes registers all built-in probe tools.
func RegisterAllProbes(r ToolRegistry) {
	probes := []Tool{
		NewProbeDisk(),
		NewProbeLargeFiles(),
		NewProbeProcess(),
		NewProbeTop(),
		NewProbeMemory(),
		NewProbeNetworkConnections(),
		NewProbeNetworkInterfaces(),
		NewProbeLogsJournal(),
		NewProbeLogsFile(),
		NewProbeServiceStatus(),
		NewProbeFileHolders(),
		NewProbeSystemInfo(),
		NewBashTool(),
		NewFileViewTool(),
		NewReadToolOutputTool(),
	}
	for _, p := range probes {
		if err := r.Register(p); err != nil {
			log.Printf("[tools] register probe %s failed: %v", p.Name(), err)
		}
	}
}

// RegisterWriteTools registers all controlled write tools (Task 12.2).
func RegisterWriteTools(r ToolRegistry) {
	writes := []Tool{
		&ServiceControlTool{},
		&TruncateLogTool{},
		&DeleteFileTool{},
		&VacuumJournalTool{},
		&LogrotateTool{},
		&KillProcessTool{},
	}
	for _, w := range writes {
		if err := r.Register(w); err != nil {
			log.Printf("[tools] register write tool %s failed: %v", w.Name(), err)
		}
	}
}

// RegisterMultiAgentTool registers the multi-agent orchestration tool.
// The executor is provided by the agent package (dependency injection).
func RegisterMultiAgentTool(r ToolRegistry, executor MultiAgentExecutor) {
	if err := r.Register(NewMultiAgentTool(executor)); err != nil {
		log.Printf("[tools] register multi-agent tool failed: %v", err)
	}
}

package tools

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
	}
	for _, p := range probes {
		_ = r.Register(p)
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
		_ = r.Register(w)
	}
}

// RegisterMultiAgentTool registers the multi-agent orchestration tool.
// The executor is provided by the agent package (dependency injection).
func RegisterMultiAgentTool(r ToolRegistry, executor MultiAgentExecutor) {
	_ = r.Register(NewMultiAgentTool(executor))
}

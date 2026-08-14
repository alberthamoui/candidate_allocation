import React, { createContext, ReactNode, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { GetAllocationRunState, StartAllocation, StopAllocation } from "../wailsjs/go/main/App";
import { EventsOn } from "../wailsjs/runtime/runtime";

export type SolverProgress = {
	percent: number;
	branchesResolved: string;
	totalBranches: string;
	branchesPruned: string;
	nodesVisited: number;
	prunedSubtrees: number;
	firstMinuteComplete: boolean;
	firstMinuteBranchesResolved: string;
	firstMinuteNodesVisited: number;
	secondMinuteComplete: boolean;
	secondMinuteBranchesResolved: string;
	secondMinuteNodesVisited: number;
};

type BackendAllocationRunState = {
	runId: number;
	running: boolean;
	hasResult: boolean;
	progress: SolverProgress;
	result?: any;
	error: string;
};

export const initialSolverProgress: SolverProgress = {
	percent: 0,
	branchesResolved: "0",
	totalBranches: "0",
	branchesPruned: "0",
	nodesVisited: 0,
	prunedSubtrees: 0,
	firstMinuteComplete: false,
	firstMinuteBranchesResolved: "0",
	firstMinuteNodesVisited: 0,
	secondMinuteComplete: false,
	secondMinuteBranchesResolved: "0",
	secondMinuteNodesVisited: 0,
};

type AllocationRunState = {
	result: any;
	progress: SolverProgress;
	running: boolean;
	error: string;
	start: (config: any) => Promise<void>;
	stop: () => Promise<boolean>;
};

const AllocationRunContext = createContext<AllocationRunState | null>(null);

export function AllocationRunProvider({ children }: { children: ReactNode }) {
	const [result, setResult] = useState<any>(null);
	const [progress, setProgress] = useState<SolverProgress>(initialSolverProgress);
	const [running, setRunning] = useState(false);
	const [error, setError] = useState("");

	useEffect(() => {
		const stopProgress = EventsOn("allocation:progress", (snapshot: SolverProgress) => setProgress(snapshot));
		const stopSolution = EventsOn("allocation:solution", (solution: any) => {
			setResult(solution);
			setRunning(true);
		});
		const stopComplete = EventsOn("allocation:complete", (solution: any) => {
			setResult(solution);
			setRunning(false);
		});
		const stopError = EventsOn("allocation:error", (message: string) => {
			setError(String(message));
			setRunning(false);
		});
		return () => {
			stopProgress();
			stopSolution();
			stopComplete();
			stopError();
		};
	}, []);

	useEffect(() => {
		let active = true;
		let requestInFlight = false;
		const synchronize = async (onlyIfBackendIsRunning: boolean) => {
			if (requestInFlight) return;
			requestInFlight = true;
			try {
				let snapshot = await GetAllocationRunState(false) as BackendAllocationRunState;
				if (!active || !snapshot || (onlyIfBackendIsRunning && !snapshot.running)) return;
				if (snapshot.hasResult) {
					const detailedSnapshot = await GetAllocationRunState(true) as BackendAllocationRunState;
					if (!active) return;
					snapshot = detailedSnapshot;
				}
				setProgress(snapshot.progress || initialSolverProgress);
				setRunning(Boolean(snapshot.running));
				setError(snapshot.error || "");
				if (snapshot.hasResult && snapshot.result) setResult(snapshot.result);
			} catch {
				// Event streaming remains the primary path. The query is only a
				// recovery mechanism for a missed final event or a page reload.
			} finally {
				requestInFlight = false;
			}
		};

		void synchronize(true);
		if (!running) return () => { active = false; };
		const interval = window.setInterval(() => void synchronize(false), 2000);
		return () => {
			active = false;
			window.clearInterval(interval);
		};
	}, [running]);

	const start = useCallback(async (config: any) => {
		setResult(null);
		setProgress(initialSolverProgress);
		setError("");
		setRunning(true);
		try {
			await StartAllocation(config);
		} catch (startError) {
			setRunning(false);
			throw startError;
		}
	}, []);

	const stop = useCallback(async () => StopAllocation(), []);
	const value = useMemo(() => ({ result, progress, running, error, start, stop }), [result, progress, running, error, start, stop]);
	return <AllocationRunContext.Provider value={value}>{children}</AllocationRunContext.Provider>;
}

export function useAllocationRun() {
	const value = useContext(AllocationRunContext);
	if (!value) {
		throw new Error("useAllocationRun deve ser usado dentro de AllocationRunProvider");
	}
	return value;
}

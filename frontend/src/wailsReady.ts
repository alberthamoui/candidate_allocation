declare global {
	interface Window {
		go?: {
			main?: {
				App?: Record<string, unknown>;
			};
		};
	}
}

function hasWailsAppBindings() {
	return typeof window !== "undefined" && !!window.go?.main?.App;
}

export async function waitForWailsBindings(timeoutMs = 5000) {
	if (hasWailsAppBindings()) {
		return;
	}

	const startedAt = Date.now();
	while (Date.now() - startedAt < timeoutMs) {
		await new Promise((resolve) => window.setTimeout(resolve, 50));
		if (hasWailsAppBindings()) {
			return;
		}
	}

	throw new Error("Bindings do Wails ainda não estão disponíveis.");
}

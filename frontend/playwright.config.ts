import { defineConfig, devices } from "@playwright/test";

// Porta própria para não conflitar com um server.exe rodando na 8080.
const PORTA = 8099;

export default defineConfig({
	testDir: "./e2e",
	timeout: 60_000,
	expect: { timeout: 10_000 },
	fullyParallel: true,
	reporter: [["list"], ["html", { open: "never" }]],
	use: {
		baseURL: `http://localhost:${PORTA}`,
		trace: "retain-on-failure",
		screenshot: "only-on-failure",
	},
	projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
	// Testa o app como ele é entregue: frontend compilado e embutido no
	// servidor Go. Nunca reaproveita um servidor já aberto, para não testar
	// código antigo.
	webServer: {
		command: "npm run build && cd .. && go run .",
		url: `http://localhost:${PORTA}/api/versao`,
		env: { PORT: String(PORTA) },
		reuseExistingServer: false,
		timeout: 180_000,
		stdout: "ignore",
		stderr: "pipe",
	},
});

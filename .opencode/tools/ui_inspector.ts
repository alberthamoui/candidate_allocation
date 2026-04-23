import { tool } from "@opencode-ai/plugin";
import path from "path";
import { exec } from "child_process";
import { promisify } from "util";

const execAsync = promisify(exec);

export default tool({
  description: "Inspeciona visualmente e estruturalmente o front-end (UI). Use para validar layouts, acessibilidade e renderização. Requer que 'wails dev' esteja rodando localmente.",
  args: {
    route: tool.schema.string().default("/").describe("A rota do front-end a ser acessada (ex: /, /config)"),
    click: tool.schema.string().optional().describe("Seletor CSS para clicar antes de capturar"),
    fill: tool.schema.string().optional().describe("No formato 'seletor:texto' para preencher um input antes de capturar"),
    wait: tool.schema.number().optional().describe("Milissegundos para esperar a tela estabilizar"),
  },
  async execute(args, context) {
    const script = path.join(context.worktree, "tools/ui-testing/agent_inspect.js");
    const outPrefix = `/tmp/agent_inspect_${Date.now()}`;
    
    // Constrói os parâmetros do comando
    let cmd = `node ${script} --route="${args.route}" --out-prefix="${outPrefix}"`;
    if (args.click) cmd += ` --click="${args.click}"`;
    if (args.fill) cmd += ` --fill="${args.fill}"`;
    if (args.wait) cmd += ` --wait="${args.wait}"`;

    try {
      // Executa o script do playwright
      const { stdout } = await execAsync(cmd);
      
      return `
✅ Inspeção concluída com sucesso!
Logs da execução:
${stdout}

Os artefatos foram salvos. Você (IA) deve usar a sua ferramenta 'Read' para analisar:
1. Imagem: ${outPrefix}.png
2. Árvore de Acessibilidade: ${outPrefix}_a11y.json
      `.trim();
    } catch (error) {
      return `❌ Erro ao acessar o Front-end. O 'wails dev' está rodando no porto 34115? 
Erro original:
${error.stdout || ''}
${error.stderr || error.message || error}`;
    }
  },
});

const fs = require('fs');

const files = [
  'frontend/src/AllocationConfigPage.tsx',
  'frontend/src/AllocationResultPage.tsx',
  'frontend/src/MappingEditorPage.tsx'
];

for (const file of files) {
  let content = fs.readFileSync(file, 'utf8');

  // Fix imports
  if (file === 'frontend/src/AllocationConfigPage.tsx') {
      content = content.replace('HelpHint,', '');
      content = content.replace('import {', 'import { HelpIcon } from "./components/Tooltip";\nimport {');
  } else if (file === 'frontend/src/AllocationResultPage.tsx') {
      content = content.replace('HelpHint,', '');
      content = content.replace('import {', 'import { HelpIcon } from "./components/Tooltip";\nimport {');
  } else if (file === 'frontend/src/MappingEditorPage.tsx') {
      content = content.replace('HelpHint,', '');
      content = content.replace('import {', 'import { HelpIcon } from "./components/Tooltip";\nimport {');
  }

  // Very careful replacements
  content = content.replace(/<HelpHint\s+label="([^"]+)"\s+content=\{([^}]+)\}\s*\/>/g, '<HelpIcon text={$2} />');
  content = content.replace(/<HelpHint\s+label=\{([^}]+)\}\s+content="([^"]+)"\s*\/>/g, '<HelpIcon text="$2" />');
  content = content.replace(/<HelpHint\s+label="([^"]+)"\s+content="([^"]+)"\s*\/>/g, '<HelpIcon text="$2" />');

  fs.writeFileSync(file, content);
}

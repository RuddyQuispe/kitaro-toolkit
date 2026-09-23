import './theme/theme.css';
import './style.css';
import './app.css';

import { renderShell } from './components/shell';

renderShell(document.querySelector<HTMLDivElement>('#app')!);

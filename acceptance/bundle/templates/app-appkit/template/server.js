import { createApp{{range $name, $_ := .plugins}}, {{$name}}{{end}} } from '@databricks/appkit';

createApp({
  plugins: [{{range $name, $_ := .plugins}}{{$name}}(),{{end}}],
}).catch(console.error);

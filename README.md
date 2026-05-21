> ⚠️ **DEPRECATED & ARCHIVED** — Migra a [`@osdo/cli`](https://github.com/opensecdevops/osdo-cli) (TypeScript/oclif)
>
> ```bash
> npm install -g @osdo/cli
> ```
>
> Todos los comandos de este CLI (Go/Cobra) existen en `@osdo/cli` con funcionalidades adicionales
> incluyendo integración con OSDO App, packages Handlebars, y despliegue basado en templates.

---

# osdo-infra-cli (Go — Archived)

CLI de infraestructura OSDO en Go/Cobra. **Reemplazado completamente por [`@osdo/cli`](https://github.com/opensecdevops/osdo-cli).**

## Guía de Migración

| Comando Go | Comando TypeScript | Notas |
|------------|-------------------|-------|
| `osdo init` | `osdo init` | Ahora soporta templates del App |
| `osdo scan` | `osdo scan` | Mismas herramientas + config mejorada |
| `osdo deploy` | `osdo deploy` | + soporte para packages del App |
| `osdo monitor` | `osdo monitor` | Idéntico |
| `osdo pipeline` | `osdo pipeline` | + sincronización con App |
| `osdo certify` | `osdo certify` | + mapeo OpenSSF/SLSA mejorado |
| *(no existía)* | `osdo app login/pull/push` | **Nuevo**: integración con OSDO App |

Para la documentación completa: [opensecdevops/osdo](https://github.com/opensecdevops/osdo)

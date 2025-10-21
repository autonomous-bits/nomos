# Nomos

Nomos is a configuration scripting language aimed to reduce configuration toil by promoting re-use and cascading overrides. Nomos aims to:

- Allow users to compose configuration by importing configuration layers, where each layer would replace any conflicting configuration values.
- Allow users to group configuration into cohesive environments.
- Allow users to reference configuration across configuration groups (environments)

These configuration scripts will be compiled producing a versioned snapshot that will be used as inputs for infrastructure as code.

## Scripting Language

The scripting language supports the following keywords:

| Keyword | Description |
| :-| :- |
| `source` | A configurable source provider, at a minimum you should be able to provide an alias and the type of provider. |
| `import` | Using an source, configuration could be imported i.e. when compiled those values should be part of a snapshot. Syntax should be `import:{alias}` or `import:{alias}:{path_to_map}`. If two or more files have conflicting properties the last import will override the previous properties. |
| `reference` | Using a source, load a specific value from the configuration. Syntax should be `reference:{alias}:{path_to_property}` |

## Tooling

A command line interface (CLI) will be provided where a script or set of scripts could be provided as inputs and then compiled. The compilation will produce a snapshot of the configuration as the output.

The CLI has one command called `build` which accepts a `--path, -p` argument to a file or folder and a `--format, -f` argument that specifies the output format `JSON, YAML, HCL` 
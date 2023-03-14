<!-- DO NOT EDIT | GENERATED CONTENT -->

# update

Will update and start a given workspace if it is out of date.

## Usage

```console
update <workspace>
```

## Description

```console
Will update and start a given workspace if it is out of date. Use --always-prompt to change the parameter values of the workspace.
```

## Options

### --always-prompt

|         |                    |
| ------- | ------------------ |
| Default | <code>false</code> |

Always prompt all parameters. Does not pull parameter values from existing workspace

### --parameter-file

|             |                                    |
| ----------- | ---------------------------------- |
| Environment | <code>$CODER_PARAMETER_FILE</code> |

Specify a file path with parameter values.

### --rich-parameter-file

|             |                                         |
| ----------- | --------------------------------------- |
| Environment | <code>$CODER_RICH_PARAMETER_FILE</code> |

Specify a file path with values for rich parameters defined in the template.
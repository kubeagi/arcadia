# RBAC Command Line Tool

This command line tool is designed to perform intelligent assessment of RBAC (Role-Based Access Control) using ZhiPuAI.

## Build

```shell
cd examples/rbac
go build .
```

## Usage

```shell
./rbac [usage]
```

### Persistent Arguments

- `--apiKey`**(Must required)**: API key to access the LLM service.
- `--model`: Model to use (default: chatglm_lite). Possible values are chatglm_lite, chatglm_std, chatglm_pro.
- `--method`: Calling method to access the LLM service (default: sse-invoke). Possible values are invoke, sse-invoke.

## Commands

### inquiry

Perform RBAC inquiry and evaluate security issues.

```shell
./rbac inquiry [args] 
```

#### Arguments

- `-f, --file`**(Must required)**: RBAC file to be queried.
- `-l, --language`: Language for the response (default: English). Can be any language! Chinese, Japanese, Korean, etc.
- `-o, --output-format`: OutputFormat for AI response.Can be default,json.

Note: The `file` argument is required.

## Examples

1. Run RBAC inquiry:

```shell
./rbac inquiry -f rbac_file.yaml --apiKey xxx 
```

Please note:

- RBAC file must be provided using the `-f` or `--file` option.
- Optional language parameter can be specified using the `-l` or `--language` option, defaulting to English.

I hope this translation meets your requirements. If you have any further questions, please feel free to ask.

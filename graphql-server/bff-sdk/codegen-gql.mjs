#!/usr/bin/env zx

import * as fs from 'fs';

const NM_RUN_PATH = 'node_modules/.bin/';
const SDK_FILE_PATH = './src/sdk.ts';

await $`${NM_RUN_PATH}graphql-codegen`;

fs.writeFileSync(
  SDK_FILE_PATH,
  fs
    .readFileSync(SDK_FILE_PATH)
    .toString()
    .replace(
      'import useSWR,',
      `import useSWR from './useSWR';
import`,
    )
    .replace(/graphql\-request\/dist\//g, 'graphql-request/src/')
    .replace(
      /\s*id:\sstring,\s*fieldName:\s*keyof\s*Variables,\s*fieldValue:\s*Variables\[\s*typeof\sfieldName\s*\]\s*/,
      `[
          id,
          fieldName,
          fieldValue
        ]: [SWRKeyInterface, keyof Variables, Variables[keyof Variables]]`,
    ),
);

await $`${NM_RUN_PATH}prettier --write \"./**/*.ts\"`;

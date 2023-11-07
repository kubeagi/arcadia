# @tenx-ui/arcadia-bff-sdk

The front-end can use this SDK to call the API of the arcadia server.

[![NPM version](https://img.shields.io/npm/v/@tenx-ui/arcadia-bff-sdk.svg?style=flat)](https://npmjs.org/package/@tenx-ui/arcadia-bff-sdk)
[![NPM downloads](http://img.shields.io/npm/dm/@tenx-ui/arcadia-bff-sdk.svg?style=flat)](https://npmjs.org/package/@tenx-ui/arcadia-bff-sdk)


## Usage

```bash
ni swr @tenx-ui/component-store-bff-client
```

```tsx
import { sdk, useSWR } from '@tenx-ui/component-store-bff-client';
import { Button, Table } from 'antd';

const Page = () => {
  const { data, loading, mute } = sdk.useGetApps();

  return (
    <div>
      <div>
        <Button type="primary" onClick={mute}>
          Refresh
        </Button>
      </div>
      <Table
        columns={[
          {
            dataIndex: 'id',
            sorter: true,
            title: 'ID',
          },
          {
            dataIndex: 'name',
            sorter: true,
            title: 'Name',
          },
          {
            dataIndex: 'description',
            sorter: true,
            title: 'Description',
          },
        ]}
        dataSource={data?.apps || []}
        loading={loading}
        rowKey="id"
      />
    </div>
  );
};
export default Page;
```

## Expansion

When the built-in methods of the SDK cannot meet the requirements, we can directly use the client for custom usage.

For example, if we only want to retrieve the ID and name of the application when obtaining the application list, we can write it like this:

```tsx
import { client, genKey } from '@tenx-ui/component-store-bff-client';
import gql from 'graphql-tag';

const GetAppsCustomDocument = gql`
  query getAppsIdName {
    apps {
      id
      name
    }
  }
`;
const getAppsCustom = () => client.request(GetAppsCustomDocument);
const useGetAppsCustom = (variables, config) =>
  useSWR(genKey('GetAppsCustom', variables), () => getAppsCustom(variables), config);
```

## Develop

Environment requirements:

- **Node.js v18.x**
- **pnpm v8.x**

Install

```bash
$ npm i pnpm @antfu/ni -g
$ ni
```

Run

```bash
$ nr dev
```

Build

```bash
$ nr build
```

Publish

```bash
$ nr pub
```

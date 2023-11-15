import { GraphQLClient } from "graphql-request";
import { GraphQLClientRequestHeaders } from "graphql-request/build/cjs/types";
import gql from "graphql-tag";
import { ClientError } from "graphql-request/src/types";
import useSWR from "./useSWR";
import {
  SWRConfiguration as SWRConfigInterface,
  Key as SWRKeyInterface,
} from "swr";
import useSWRInfinite, { SWRInfiniteConfiguration } from "swr/infinite";
export type Maybe<T> = T | null;
export type InputMaybe<T> = Maybe<T>;
export type Exact<T extends { [key: string]: unknown }> = {
  [K in keyof T]: T[K];
};
export type MakeOptional<T, K extends keyof T> = Omit<T, K> & {
  [SubKey in K]?: Maybe<T[SubKey]>;
};
export type MakeMaybe<T, K extends keyof T> = Omit<T, K> & {
  [SubKey in K]: Maybe<T[SubKey]>;
};
export type MakeEmpty<
  T extends { [key: string]: unknown },
  K extends keyof T,
> = { [_ in K]?: never };
export type Incremental<T> =
  | T
  | {
      [P in keyof T]?: P extends " $fragmentName" | "__typename" ? T[P] : never;
    };
/** All built-in and custom scalars, mapped to their actual values */
export type Scalars = {
  ID: { input: string; output: string };
  String: { input: string; output: string };
  Boolean: { input: boolean; output: boolean };
  Int: { input: number; output: number };
  Float: { input: number; output: number };
  Map: { input: any; output: any };
  Time: { input: any; output: any };
  Void: { input: any; output: any };
};

export type CreateDatasourceInput = {
  annotations?: InputMaybe<Scalars["Map"]["input"]>;
  description?: InputMaybe<Scalars["String"]["input"]>;
  displayName: Scalars["String"]["input"];
  endpointinput?: InputMaybe<EndpointInput>;
  labels?: InputMaybe<Scalars["Map"]["input"]>;
  name: Scalars["String"]["input"];
  namespace: Scalars["String"]["input"];
  ossinput?: InputMaybe<OssInput>;
};

export type Datasource = {
  __typename?: "Datasource";
  annotations?: Maybe<Scalars["Map"]["output"]>;
  creator?: Maybe<Scalars["String"]["output"]>;
  displayName: Scalars["String"]["output"];
  endpoint?: Maybe<Endpoint>;
  fileCount?: Maybe<Scalars["Int"]["output"]>;
  labels?: Maybe<Scalars["Map"]["output"]>;
  name: Scalars["String"]["output"];
  namespace: Scalars["String"]["output"];
  oss?: Maybe<Oss>;
  status?: Maybe<Scalars["Boolean"]["output"]>;
};

export type DeleteDatasourceInput = {
  fieldSelector?: InputMaybe<Scalars["String"]["input"]>;
  labelSelector?: InputMaybe<Scalars["String"]["input"]>;
  name?: InputMaybe<Scalars["String"]["input"]>;
  namespace: Scalars["String"]["input"];
};

export type Endpoint = {
  __typename?: "Endpoint";
  authSecret?: Maybe<TypedObjectReference>;
  insecure?: Maybe<Scalars["Boolean"]["output"]>;
  url?: Maybe<Scalars["String"]["output"]>;
};

export type EndpointInput = {
  authSecret?: InputMaybe<TypedObjectReferenceInput>;
  insecure?: InputMaybe<Scalars["Boolean"]["input"]>;
  url?: InputMaybe<Scalars["String"]["input"]>;
};

export type ListDatasourceInput = {
  displayName?: InputMaybe<Scalars["String"]["input"]>;
  fieldSelector?: InputMaybe<Scalars["String"]["input"]>;
  keyword?: InputMaybe<Scalars["String"]["input"]>;
  labelSelector?: InputMaybe<Scalars["String"]["input"]>;
  name?: InputMaybe<Scalars["String"]["input"]>;
  namespace: Scalars["String"]["input"];
  page?: InputMaybe<Scalars["Int"]["input"]>;
  pageSize?: InputMaybe<Scalars["Int"]["input"]>;
};

export type Mutation = {
  __typename?: "Mutation";
  createDatasource: Datasource;
  deleteDatasource?: Maybe<Scalars["Void"]["output"]>;
  updateDatasource: Datasource;
};

export type MutationCreateDatasourceArgs = {
  input: CreateDatasourceInput;
};

export type MutationDeleteDatasourceArgs = {
  input?: InputMaybe<DeleteDatasourceInput>;
};

export type MutationUpdateDatasourceArgs = {
  input?: InputMaybe<UpdateDatasourceInput>;
};

export type Oss = {
  __typename?: "Oss";
  Object?: Maybe<Scalars["String"]["output"]>;
  bucket?: Maybe<Scalars["String"]["output"]>;
};

export type OssInput = {
  Object?: InputMaybe<Scalars["String"]["input"]>;
  bucket?: InputMaybe<Scalars["String"]["input"]>;
};

export type PaginatedDatasource = {
  __typename?: "PaginatedDatasource";
  hasNextPage: Scalars["Boolean"]["output"];
  nodes?: Maybe<Array<Datasource>>;
  page?: Maybe<Scalars["Int"]["output"]>;
  pageSize?: Maybe<Scalars["Int"]["output"]>;
  totalCount: Scalars["Int"]["output"];
};

export type Query = {
  __typename?: "Query";
  datasource: Datasource;
  datasourcesPaged: PaginatedDatasource;
};

export type QueryDatasourceArgs = {
  name: Scalars["String"]["input"];
  namespace: Scalars["String"]["input"];
};

export type QueryDatasourcesPagedArgs = {
  input: ListDatasourceInput;
};

export type TypedObjectReference = {
  __typename?: "TypedObjectReference";
  Name: Scalars["String"]["output"];
  Namespace?: Maybe<Scalars["String"]["output"]>;
  apiGroup?: Maybe<Scalars["String"]["output"]>;
  kind: Scalars["String"]["output"];
};

export type TypedObjectReferenceInput = {
  Name: Scalars["String"]["input"];
  Namespace?: InputMaybe<Scalars["String"]["input"]>;
  apiGroup?: InputMaybe<Scalars["String"]["input"]>;
  kind: Scalars["String"]["input"];
};

export type UpdateDatasourceInput = {
  annotations?: InputMaybe<Scalars["Map"]["input"]>;
  description?: InputMaybe<Scalars["String"]["input"]>;
  displayName: Scalars["String"]["input"];
  labels?: InputMaybe<Scalars["Map"]["input"]>;
  name: Scalars["String"]["input"];
  namespace: Scalars["String"]["input"];
};

export type CreateDatasourceMutationVariables = Exact<{
  input: CreateDatasourceInput;
}>;

export type CreateDatasourceMutation = {
  __typename?: "Mutation";
  createDatasource: {
    __typename?: "Datasource";
    name: string;
    namespace: string;
    displayName: string;
    endpoint?: {
      __typename?: "Endpoint";
      url?: string | null;
      insecure?: boolean | null;
      authSecret?: {
        __typename?: "TypedObjectReference";
        kind: string;
        Name: string;
      } | null;
    } | null;
    oss?: { __typename?: "Oss"; bucket?: string | null } | null;
  };
};

export type UpdateDatasourceMutationVariables = Exact<{
  input?: InputMaybe<UpdateDatasourceInput>;
}>;

export type UpdateDatasourceMutation = {
  __typename?: "Mutation";
  updateDatasource: {
    __typename?: "Datasource";
    name: string;
    namespace: string;
    displayName: string;
    endpoint?: {
      __typename?: "Endpoint";
      url?: string | null;
      insecure?: boolean | null;
      authSecret?: {
        __typename?: "TypedObjectReference";
        kind: string;
        Name: string;
      } | null;
    } | null;
    oss?: { __typename?: "Oss"; bucket?: string | null } | null;
  };
};

export type DeleteDatasourceMutationVariables = Exact<{
  input: DeleteDatasourceInput;
}>;

export type DeleteDatasourceMutation = {
  __typename?: "Mutation";
  deleteDatasource?: any | null;
};

export type GetDatasourcesPagedQueryVariables = Exact<{
  input: ListDatasourceInput;
}>;

export type GetDatasourcesPagedQuery = {
  __typename?: "Query";
  datasourcesPaged: {
    __typename?: "PaginatedDatasource";
    totalCount: number;
    hasNextPage: boolean;
    nodes?: Array<{
      __typename?: "Datasource";
      name: string;
      namespace: string;
      displayName: string;
      endpoint?: {
        __typename?: "Endpoint";
        url?: string | null;
        insecure?: boolean | null;
        authSecret?: {
          __typename?: "TypedObjectReference";
          kind: string;
          Name: string;
        } | null;
      } | null;
      oss?: { __typename?: "Oss"; bucket?: string | null } | null;
    }> | null;
  };
};

export type GetDatasourceQueryVariables = Exact<{
  name: Scalars["String"]["input"];
  namespace: Scalars["String"]["input"];
}>;

export type GetDatasourceQuery = {
  __typename?: "Query";
  datasource: {
    __typename?: "Datasource";
    name: string;
    namespace: string;
    displayName: string;
    endpoint?: {
      __typename?: "Endpoint";
      url?: string | null;
      insecure?: boolean | null;
      authSecret?: {
        __typename?: "TypedObjectReference";
        kind: string;
        Name: string;
      } | null;
    } | null;
    oss?: { __typename?: "Oss"; bucket?: string | null } | null;
  };
};

export const CreateDatasourceDocument = gql`
  mutation createDatasource($input: CreateDatasourceInput!) {
    createDatasource(input: $input) {
      name
      namespace
      displayName
      endpoint {
        url
        authSecret {
          kind
          Name
        }
        insecure
      }
      oss {
        bucket
      }
    }
  }
`;
export const UpdateDatasourceDocument = gql`
  mutation updateDatasource($input: UpdateDatasourceInput) {
    updateDatasource(input: $input) {
      name
      namespace
      displayName
      endpoint {
        url
        authSecret {
          kind
          Name
        }
        insecure
      }
      oss {
        bucket
      }
    }
  }
`;
export const DeleteDatasourceDocument = gql`
  mutation deleteDatasource($input: DeleteDatasourceInput!) {
    deleteDatasource(input: $input)
  }
`;
export const GetDatasourcesPagedDocument = gql`
  query getDatasourcesPaged($input: ListDatasourceInput!) {
    datasourcesPaged(input: $input) {
      totalCount
      hasNextPage
      nodes {
        name
        namespace
        displayName
        endpoint {
          url
          authSecret {
            kind
            Name
          }
          insecure
        }
        oss {
          bucket
        }
      }
    }
  }
`;
export const GetDatasourceDocument = gql`
  query getDatasource($name: String!, $namespace: String!) {
    datasource(name: $name, namespace: $namespace) {
      name
      namespace
      displayName
      endpoint {
        url
        authSecret {
          kind
          Name
        }
        insecure
      }
      oss {
        bucket
      }
    }
  }
`;

export type SdkFunctionWrapper = <T>(
  action: (requestHeaders?: Record<string, string>) => Promise<T>,
  operationName: string,
  operationType?: string,
) => Promise<T>;

const defaultWrapper: SdkFunctionWrapper = (
  action,
  _operationName,
  _operationType,
) => action();

export function getSdk(
  client: GraphQLClient,
  withWrapper: SdkFunctionWrapper = defaultWrapper,
) {
  return {
    createDatasource(
      variables: CreateDatasourceMutationVariables,
      requestHeaders?: GraphQLClientRequestHeaders,
    ): Promise<CreateDatasourceMutation> {
      return withWrapper(
        (wrappedRequestHeaders) =>
          client.request<CreateDatasourceMutation>(
            CreateDatasourceDocument,
            variables,
            { ...requestHeaders, ...wrappedRequestHeaders },
          ),
        "createDatasource",
        "mutation",
      );
    },
    updateDatasource(
      variables?: UpdateDatasourceMutationVariables,
      requestHeaders?: GraphQLClientRequestHeaders,
    ): Promise<UpdateDatasourceMutation> {
      return withWrapper(
        (wrappedRequestHeaders) =>
          client.request<UpdateDatasourceMutation>(
            UpdateDatasourceDocument,
            variables,
            { ...requestHeaders, ...wrappedRequestHeaders },
          ),
        "updateDatasource",
        "mutation",
      );
    },
    deleteDatasource(
      variables: DeleteDatasourceMutationVariables,
      requestHeaders?: GraphQLClientRequestHeaders,
    ): Promise<DeleteDatasourceMutation> {
      return withWrapper(
        (wrappedRequestHeaders) =>
          client.request<DeleteDatasourceMutation>(
            DeleteDatasourceDocument,
            variables,
            { ...requestHeaders, ...wrappedRequestHeaders },
          ),
        "deleteDatasource",
        "mutation",
      );
    },
    getDatasourcesPaged(
      variables: GetDatasourcesPagedQueryVariables,
      requestHeaders?: GraphQLClientRequestHeaders,
    ): Promise<GetDatasourcesPagedQuery> {
      return withWrapper(
        (wrappedRequestHeaders) =>
          client.request<GetDatasourcesPagedQuery>(
            GetDatasourcesPagedDocument,
            variables,
            { ...requestHeaders, ...wrappedRequestHeaders },
          ),
        "getDatasourcesPaged",
        "query",
      );
    },
    getDatasource(
      variables: GetDatasourceQueryVariables,
      requestHeaders?: GraphQLClientRequestHeaders,
    ): Promise<GetDatasourceQuery> {
      return withWrapper(
        (wrappedRequestHeaders) =>
          client.request<GetDatasourceQuery>(GetDatasourceDocument, variables, {
            ...requestHeaders,
            ...wrappedRequestHeaders,
          }),
        "getDatasource",
        "query",
      );
    },
  };
}
export type Sdk = ReturnType<typeof getSdk>;
export type SWRInfiniteKeyLoader<Data = unknown, Variables = unknown> = (
  index: number,
  previousPageData: Data | null,
) => [keyof Variables, Variables[keyof Variables] | null] | null;
export function getSdkWithHooks(
  client: GraphQLClient,
  withWrapper: SdkFunctionWrapper = defaultWrapper,
) {
  const sdk = getSdk(client, withWrapper);
  const utilsForInfinite = {
    generateGetKey:
      <Data = unknown, Variables = unknown>(
        id: SWRKeyInterface,
        getKey: SWRInfiniteKeyLoader<Data, Variables>,
      ) =>
      (pageIndex: number, previousData: Data | null) => {
        const key = getKey(pageIndex, previousData);
        return key ? [id, ...key] : null;
      },
    generateFetcher:
      <Query = unknown, Variables = unknown>(
        query: (variables: Variables) => Promise<Query>,
        variables?: Variables,
      ) =>
      ([id, fieldName, fieldValue]: [
        SWRKeyInterface,
        keyof Variables,
        Variables[keyof Variables],
      ]) =>
        query({ ...variables, [fieldName]: fieldValue } as Variables),
  };
  const genKey = <V extends Record<string, unknown> = Record<string, unknown>>(
    name: string,
    object: V = {} as V,
  ): SWRKeyInterface => [
    name,
    ...Object.keys(object)
      .sort()
      .map((key) => object[key]),
  ];
  return {
    ...sdk,
    useGetDatasourcesPaged(
      variables: GetDatasourcesPagedQueryVariables,
      config?: SWRConfigInterface<GetDatasourcesPagedQuery, ClientError>,
    ) {
      return useSWR<GetDatasourcesPagedQuery, ClientError>(
        genKey<GetDatasourcesPagedQueryVariables>(
          "GetDatasourcesPaged",
          variables,
        ),
        () => sdk.getDatasourcesPaged(variables),
        config,
      );
    },
    useGetDatasource(
      variables: GetDatasourceQueryVariables,
      config?: SWRConfigInterface<GetDatasourceQuery, ClientError>,
    ) {
      return useSWR<GetDatasourceQuery, ClientError>(
        genKey<GetDatasourceQueryVariables>("GetDatasource", variables),
        () => sdk.getDatasource(variables),
        config,
      );
    },
  };
}
export type SdkWithHooks = ReturnType<typeof getSdkWithHooks>;

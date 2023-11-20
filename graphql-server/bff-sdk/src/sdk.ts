import { GraphQLClient } from "graphql-request";
import { GraphQLClientRequestHeaders } from "graphql-request/build/cjs/types";
import gql from "graphql-tag";
import { ClientError } from "graphql-request/src/types";
import useSWR from "./useSWR";
import {
  SWRConfiguration as SWRConfigInterface,
  Key as SWRKeyInterface,
} from "swr";
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

export type CreateModelInput = {
  description?: InputMaybe<Scalars["String"]["input"]>;
  displayName: Scalars["String"]["input"];
  field: Scalars["String"]["input"];
  modeltype: Scalars["String"]["input"];
  name: Scalars["String"]["input"];
  namespace: Scalars["String"]["input"];
  updateTimestamp?: InputMaybe<Scalars["Time"]["input"]>;
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
  updateTimestamp: Scalars["Time"]["output"];
};

export type DatasourceMuation = {
  __typename?: "DatasourceMuation";
  createDatasource: Datasource;
  deleteDatasource?: Maybe<Scalars["Void"]["output"]>;
  updateDatasource: Datasource;
};

export type DatasourceMuationCreateDatasourceArgs = {
  input: CreateDatasourceInput;
};

export type DatasourceMuationDeleteDatasourceArgs = {
  input?: InputMaybe<DeleteDatasourceInput>;
};

export type DatasourceMuationUpdateDatasourceArgs = {
  input?: InputMaybe<UpdateDatasourceInput>;
};

export type DatasourceQuery = {
  __typename?: "DatasourceQuery";
  getDatasource: Datasource;
  listDatasources: PaginatedResult;
};

export type DatasourceQueryGetDatasourceArgs = {
  name: Scalars["String"]["input"];
  namespace: Scalars["String"]["input"];
};

export type DatasourceQueryListDatasourcesArgs = {
  input: ListDatasourceInput;
};

export type DeleteDatasourceInput = {
  fieldSelector?: InputMaybe<Scalars["String"]["input"]>;
  labelSelector?: InputMaybe<Scalars["String"]["input"]>;
  name?: InputMaybe<Scalars["String"]["input"]>;
  namespace: Scalars["String"]["input"];
};

export type DeleteModelInput = {
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

export type ListModelInput = {
  displayName?: InputMaybe<Scalars["String"]["input"]>;
  fieldSelector?: InputMaybe<Scalars["String"]["input"]>;
  keyword?: InputMaybe<Scalars["String"]["input"]>;
  labelSelector?: InputMaybe<Scalars["String"]["input"]>;
  name?: InputMaybe<Scalars["String"]["input"]>;
  namespace: Scalars["String"]["input"];
  page?: InputMaybe<Scalars["Int"]["input"]>;
  pageSize?: InputMaybe<Scalars["Int"]["input"]>;
};

export type Model = {
  __typename?: "Model";
  annotations?: Maybe<Scalars["Map"]["output"]>;
  creator?: Maybe<Scalars["String"]["output"]>;
  description?: Maybe<Scalars["String"]["output"]>;
  displayName: Scalars["String"]["output"];
  field: Scalars["String"]["output"];
  labels?: Maybe<Scalars["Map"]["output"]>;
  modeltype: Scalars["String"]["output"];
  name: Scalars["String"]["output"];
  namespace: Scalars["String"]["output"];
  updateTimestamp?: Maybe<Scalars["Time"]["output"]>;
};

export type ModelMutation = {
  __typename?: "ModelMutation";
  createModel: Model;
  deleteModel?: Maybe<Scalars["Void"]["output"]>;
  updateModel: Model;
};

export type ModelMutationCreateModelArgs = {
  input: CreateModelInput;
};

export type ModelMutationDeleteModelArgs = {
  input?: InputMaybe<DeleteModelInput>;
};

export type ModelMutationUpdateModelArgs = {
  input?: InputMaybe<UpdateModelInput>;
};

export type ModelQuery = {
  __typename?: "ModelQuery";
  getModel: Model;
  listModels: PaginatedResult;
};

export type ModelQueryGetModelArgs = {
  name: Scalars["String"]["input"];
  namespace: Scalars["String"]["input"];
};

export type ModelQueryListModelsArgs = {
  input: ListModelInput;
};

export type Mutation = {
  __typename?: "Mutation";
  Datasource?: Maybe<DatasourceMuation>;
  Model?: Maybe<ModelMutation>;
  hello: Scalars["String"]["output"];
};

export type MutationHelloArgs = {
  name: Scalars["String"]["input"];
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

export type PageNode = Datasource | Model;

export type PaginatedResult = {
  __typename?: "PaginatedResult";
  hasNextPage: Scalars["Boolean"]["output"];
  nodes?: Maybe<Array<PageNode>>;
  page?: Maybe<Scalars["Int"]["output"]>;
  pageSize?: Maybe<Scalars["Int"]["output"]>;
  totalCount: Scalars["Int"]["output"];
};

export type Query = {
  __typename?: "Query";
  Datasource?: Maybe<DatasourceQuery>;
  Model?: Maybe<ModelQuery>;
  hello: Scalars["String"]["output"];
};

export type QueryHelloArgs = {
  name: Scalars["String"]["input"];
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

export type UpdateModelInput = {
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
  Datasource?: {
    __typename?: "DatasourceMuation";
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
  } | null;
};

export type UpdateDatasourceMutationVariables = Exact<{
  input?: InputMaybe<UpdateDatasourceInput>;
}>;

export type UpdateDatasourceMutation = {
  __typename?: "Mutation";
  Datasource?: {
    __typename?: "DatasourceMuation";
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
  } | null;
};

export type DeleteDatasourceMutationVariables = Exact<{
  input: DeleteDatasourceInput;
}>;

export type DeleteDatasourceMutation = {
  __typename?: "Mutation";
  Datasource?: {
    __typename?: "DatasourceMuation";
    deleteDatasource?: any | null;
  } | null;
};

export type ListDatasourcesQueryVariables = Exact<{
  input: ListDatasourceInput;
}>;

export type ListDatasourcesQuery = {
  __typename?: "Query";
  Datasource?: {
    __typename?: "DatasourceQuery";
    listDatasources: {
      __typename?: "PaginatedResult";
      totalCount: number;
      hasNextPage: boolean;
      nodes?: Array<
        | {
            __typename: "Datasource";
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
          }
        | { __typename: "Model" }
      > | null;
    };
  } | null;
};

export type GetDatasourceQueryVariables = Exact<{
  name: Scalars["String"]["input"];
  namespace: Scalars["String"]["input"];
}>;

export type GetDatasourceQuery = {
  __typename?: "Query";
  Datasource?: {
    __typename?: "DatasourceQuery";
    getDatasource: {
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
  } | null;
};

export type ListModelsQueryVariables = Exact<{
  input: ListModelInput;
}>;

export type ListModelsQuery = {
  __typename?: "Query";
  Model?: {
    __typename?: "ModelQuery";
    listModels: {
      __typename?: "PaginatedResult";
      totalCount: number;
      hasNextPage: boolean;
      nodes?: Array<
        | { __typename: "Datasource" }
        | {
            __typename: "Model";
            name: string;
            namespace: string;
            labels?: any | null;
            annotations?: any | null;
            creator?: string | null;
            displayName: string;
            description?: string | null;
            field: string;
            modeltype: string;
            updateTimestamp?: any | null;
          }
      > | null;
    };
  } | null;
};

export type GetModelQueryVariables = Exact<{
  name: Scalars["String"]["input"];
  namespace: Scalars["String"]["input"];
}>;

export type GetModelQuery = {
  __typename?: "Query";
  Model?: {
    __typename?: "ModelQuery";
    getModel: {
      __typename?: "Model";
      name: string;
      namespace: string;
      labels?: any | null;
      annotations?: any | null;
      creator?: string | null;
      displayName: string;
      description?: string | null;
      field: string;
      modeltype: string;
      updateTimestamp?: any | null;
    };
  } | null;
};

export type CreateModelMutationVariables = Exact<{
  input: CreateModelInput;
}>;

export type CreateModelMutation = {
  __typename?: "Mutation";
  Model?: {
    __typename?: "ModelMutation";
    createModel: {
      __typename?: "Model";
      name: string;
      namespace: string;
      labels?: any | null;
      annotations?: any | null;
      creator?: string | null;
      displayName: string;
      description?: string | null;
      field: string;
      modeltype: string;
      updateTimestamp?: any | null;
    };
  } | null;
};

export type UpdateModelMutationVariables = Exact<{
  input?: InputMaybe<UpdateModelInput>;
}>;

export type UpdateModelMutation = {
  __typename?: "Mutation";
  Model?: {
    __typename?: "ModelMutation";
    updateModel: {
      __typename?: "Model";
      name: string;
      namespace: string;
      labels?: any | null;
      annotations?: any | null;
      creator?: string | null;
      displayName: string;
      description?: string | null;
      field: string;
      modeltype: string;
      updateTimestamp?: any | null;
    };
  } | null;
};

export type DeleteModelMutationVariables = Exact<{
  input?: InputMaybe<DeleteModelInput>;
}>;

export type DeleteModelMutation = {
  __typename?: "Mutation";
  Model?: { __typename?: "ModelMutation"; deleteModel?: any | null } | null;
};

export const CreateDatasourceDocument = gql`
  mutation createDatasource($input: CreateDatasourceInput!) {
    Datasource {
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
  }
`;
export const UpdateDatasourceDocument = gql`
  mutation updateDatasource($input: UpdateDatasourceInput) {
    Datasource {
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
  }
`;
export const DeleteDatasourceDocument = gql`
  mutation deleteDatasource($input: DeleteDatasourceInput!) {
    Datasource {
      deleteDatasource(input: $input)
    }
  }
`;
export const ListDatasourcesDocument = gql`
  query listDatasources($input: ListDatasourceInput!) {
    Datasource {
      listDatasources(input: $input) {
        totalCount
        hasNextPage
        nodes {
          __typename
          ... on Datasource {
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
    }
  }
`;
export const GetDatasourceDocument = gql`
  query getDatasource($name: String!, $namespace: String!) {
    Datasource {
      getDatasource(name: $name, namespace: $namespace) {
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
export const ListModelsDocument = gql`
  query listModels($input: ListModelInput!) {
    Model {
      listModels(input: $input) {
        totalCount
        hasNextPage
        nodes {
          __typename
          ... on Model {
            name
            namespace
            labels
            annotations
            creator
            displayName
            description
            field
            modeltype
            updateTimestamp
          }
        }
      }
    }
  }
`;
export const GetModelDocument = gql`
  query getModel($name: String!, $namespace: String!) {
    Model {
      getModel(name: $name, namespace: $namespace) {
        name
        namespace
        labels
        annotations
        creator
        displayName
        description
        field
        modeltype
        updateTimestamp
      }
    }
  }
`;
export const CreateModelDocument = gql`
  mutation createModel($input: CreateModelInput!) {
    Model {
      createModel(input: $input) {
        name
        namespace
        labels
        annotations
        creator
        displayName
        description
        field
        modeltype
        updateTimestamp
      }
    }
  }
`;
export const UpdateModelDocument = gql`
  mutation updateModel($input: UpdateModelInput) {
    Model {
      updateModel(input: $input) {
        name
        namespace
        labels
        annotations
        creator
        displayName
        description
        field
        modeltype
        updateTimestamp
      }
    }
  }
`;
export const DeleteModelDocument = gql`
  mutation deleteModel($input: DeleteModelInput) {
    Model {
      deleteModel(input: $input)
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
    listDatasources(
      variables: ListDatasourcesQueryVariables,
      requestHeaders?: GraphQLClientRequestHeaders,
    ): Promise<ListDatasourcesQuery> {
      return withWrapper(
        (wrappedRequestHeaders) =>
          client.request<ListDatasourcesQuery>(
            ListDatasourcesDocument,
            variables,
            { ...requestHeaders, ...wrappedRequestHeaders },
          ),
        "listDatasources",
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
    listModels(
      variables: ListModelsQueryVariables,
      requestHeaders?: GraphQLClientRequestHeaders,
    ): Promise<ListModelsQuery> {
      return withWrapper(
        (wrappedRequestHeaders) =>
          client.request<ListModelsQuery>(ListModelsDocument, variables, {
            ...requestHeaders,
            ...wrappedRequestHeaders,
          }),
        "listModels",
        "query",
      );
    },
    getModel(
      variables: GetModelQueryVariables,
      requestHeaders?: GraphQLClientRequestHeaders,
    ): Promise<GetModelQuery> {
      return withWrapper(
        (wrappedRequestHeaders) =>
          client.request<GetModelQuery>(GetModelDocument, variables, {
            ...requestHeaders,
            ...wrappedRequestHeaders,
          }),
        "getModel",
        "query",
      );
    },
    createModel(
      variables: CreateModelMutationVariables,
      requestHeaders?: GraphQLClientRequestHeaders,
    ): Promise<CreateModelMutation> {
      return withWrapper(
        (wrappedRequestHeaders) =>
          client.request<CreateModelMutation>(CreateModelDocument, variables, {
            ...requestHeaders,
            ...wrappedRequestHeaders,
          }),
        "createModel",
        "mutation",
      );
    },
    updateModel(
      variables?: UpdateModelMutationVariables,
      requestHeaders?: GraphQLClientRequestHeaders,
    ): Promise<UpdateModelMutation> {
      return withWrapper(
        (wrappedRequestHeaders) =>
          client.request<UpdateModelMutation>(UpdateModelDocument, variables, {
            ...requestHeaders,
            ...wrappedRequestHeaders,
          }),
        "updateModel",
        "mutation",
      );
    },
    deleteModel(
      variables?: DeleteModelMutationVariables,
      requestHeaders?: GraphQLClientRequestHeaders,
    ): Promise<DeleteModelMutation> {
      return withWrapper(
        (wrappedRequestHeaders) =>
          client.request<DeleteModelMutation>(DeleteModelDocument, variables, {
            ...requestHeaders,
            ...wrappedRequestHeaders,
          }),
        "deleteModel",
        "mutation",
      );
    },
  };
}
export type Sdk = ReturnType<typeof getSdk>;
export function getSdkWithHooks(
  client: GraphQLClient,
  withWrapper: SdkFunctionWrapper = defaultWrapper,
) {
  const sdk = getSdk(client, withWrapper);
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
    useListDatasources(
      variables: ListDatasourcesQueryVariables,
      config?: SWRConfigInterface<ListDatasourcesQuery, ClientError>,
    ) {
      return useSWR<ListDatasourcesQuery, ClientError>(
        genKey<ListDatasourcesQueryVariables>("ListDatasources", variables),
        () => sdk.listDatasources(variables),
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
    useListModels(
      variables: ListModelsQueryVariables,
      config?: SWRConfigInterface<ListModelsQuery, ClientError>,
    ) {
      return useSWR<ListModelsQuery, ClientError>(
        genKey<ListModelsQueryVariables>("ListModels", variables),
        () => sdk.listModels(variables),
        config,
      );
    },
    useGetModel(
      variables: GetModelQueryVariables,
      config?: SWRConfigInterface<GetModelQuery, ClientError>,
    ) {
      return useSWR<GetModelQuery, ClientError>(
        genKey<GetModelQueryVariables>("GetModel", variables),
        () => sdk.getModel(variables),
        config,
      );
    },
  };
}
export type SdkWithHooks = ReturnType<typeof getSdkWithHooks>;

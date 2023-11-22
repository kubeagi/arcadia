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

export type CreateDatasetInput = {
  /** 一些备注用的注视信息，或者记录一个简单的配置 */
  annotations?: InputMaybe<Scalars["Map"]["input"]>;
  /** 数据集里面的数据的类型，文本，视频，图片 */
  contentType: Scalars["String"]["input"];
  /** 描述信息，可以不写 */
  description?: InputMaybe<Scalars["String"]["input"]>;
  /** 展示名称，用于展示在界面上的，必须填写 */
  displayName: Scalars["String"]["input"];
  /** 应用场景，可以为空 */
  filed?: InputMaybe<Scalars["String"]["input"]>;
  /** 一些标签选择信息，可以不添加 */
  labels?: InputMaybe<Scalars["Map"]["input"]>;
  /** 数据集的CR名字，要满足k8s的名称规则 */
  name: Scalars["String"]["input"];
  namespace: Scalars["String"]["input"];
};

/** 新增数据源时输入条件 */
export type CreateDatasourceInput = {
  /** 数据源资源注释 */
  annotations?: InputMaybe<Scalars["Map"]["input"]>;
  /** 数据源资源描述 */
  description?: InputMaybe<Scalars["String"]["input"]>;
  /** 数据源资源展示名称作为显示，并提供编辑 */
  displayName: Scalars["String"]["input"];
  /** 提供对象存储时输入条件 */
  endpointinput?: InputMaybe<EndpointInput>;
  /** 数据源资源标签 */
  labels?: InputMaybe<Scalars["Map"]["input"]>;
  /** 数据源资源名称（不可同名） */
  name: Scalars["String"]["input"];
  /** 数据源创建命名空间 */
  namespace: Scalars["String"]["input"];
  ossinput?: InputMaybe<OssInput>;
};

export type CreateEmbedderInput = {
  /** 模型服务资源注释 */
  annotations?: InputMaybe<Scalars["Map"]["input"]>;
  /** 模型服务资源描述 */
  description?: InputMaybe<Scalars["String"]["input"]>;
  /** 模型服务资源展示名称作为显示，并提供编辑 */
  displayName: Scalars["String"]["input"];
  endpointinput?: InputMaybe<EndpointInput>;
  /** 模型服务资源标签 */
  labels?: InputMaybe<Scalars["Map"]["input"]>;
  /** 模型服务资源名称（不可同名） */
  name: Scalars["String"]["input"];
  /** 模型服务创建命名空间 */
  namespace: Scalars["String"]["input"];
  /** 模型服务类型 */
  serviceType?: InputMaybe<Scalars["String"]["input"]>;
};

export type CreateKnowledgeBaseInput = {
  /** 知识库资源注释 */
  annotations?: InputMaybe<Scalars["Map"]["input"]>;
  /** 知识库资源描述 */
  description?: InputMaybe<Scalars["String"]["input"]>;
  /** 知识库资源展示名称作为显示，并提供编辑 */
  displayName: Scalars["String"]["input"];
  /** 模型服务 */
  embedder?: InputMaybe<TypedObjectReferenceInput>;
  /** 知识库文件 */
  fileGroups?: InputMaybe<Array<Filegroupinput>>;
  /** 知识库资源标签 */
  labels?: InputMaybe<Scalars["Map"]["input"]>;
  /** 知识库资源名称（不可同名） */
  name: Scalars["String"]["input"];
  /** 知识库创建命名空间 */
  namespace: Scalars["String"]["input"];
  /** "向量数据库(使用默认值) */
  vectorStore?: InputMaybe<TypedObjectReferenceInput>;
};

export type CreateModelInput = {
  /** 模型资源描述 */
  description?: InputMaybe<Scalars["String"]["input"]>;
  /** 模型资源展示名称作为显示，并提供编辑 */
  displayName: Scalars["String"]["input"];
  /** 模型应用领域 */
  field: Scalars["String"]["input"];
  /** 模型类型 */
  modeltypes: Scalars["String"]["input"];
  /** 模型资源名称（不可同名） */
  name: Scalars["String"]["input"];
  /** 模型创建命名空间 */
  namespace: Scalars["String"]["input"];
};

export type CreateVersionedDatasetInput = {
  /** 一些备注用的注视信息，或者记录一个简单的配置 */
  annotations?: InputMaybe<Scalars["Map"]["input"]>;
  /**
   * dataset的名字，需要根据这个名字，
   *     判断是否最新版本不包含任何文件(产品要求，有一个不包含任何文件的版本，不允许创建新的版本)
   */
  datasetName: Scalars["String"]["input"];
  /** 描述信息，可以不写 */
  description?: InputMaybe<Scalars["String"]["input"]>;
  /** 展示名称，用于展示在界面上的，必须填写 */
  displayName: Scalars["String"]["input"];
  /** 从数据源要上传的文件，目前以及不用了 */
  fileGrups?: InputMaybe<Array<InputMaybe<FileGroup>>>;
  /** 界面上创建新版本选择从某个版本集成的时候，填写version字段 */
  inheritedFrom?: InputMaybe<Scalars["String"]["input"]>;
  /** 一些标签选择信息，可以不添加 */
  labels?: InputMaybe<Scalars["Map"]["input"]>;
  /** 数据集的CR名字，要满足k8s的名称规则 */
  name: Scalars["String"]["input"];
  namespace: Scalars["String"]["input"];
  /** 是否发布，0是未发布，1是已经发布，创建一个版本的时候默认传递0就可以 */
  released: Scalars["Int"]["input"];
  /** 数据集里面的数据的类型，文本，视频，图片 */
  version: Scalars["String"]["input"];
};

/**
 * Dataset
 * 数据集代表用户纳管的一组相似属性的文件，采用相同的方式进行数据处理并用于后续的
 * 1. 模型训练
 * 2. 知识库
 *
 * 支持多种类型数据:
 * - 文本
 * - 图片
 * - 视频
 *
 * 单个数据集仅允许包含同一类型文件，不同类型文件将被忽略
 * 数据集允许有多个版本，数据处理针对单个版本进行
 * 数据集某个版本完成数据处理后，数据处理服务需要将处理后的存储回 版本数据集
 */
export type Dataset = {
  __typename?: "Dataset";
  /** 添加一些辅助性记录信息 */
  annotations?: Maybe<Scalars["Map"]["output"]>;
  /** 数据集类型，文本，图片，视频 */
  contentType: Scalars["String"]["output"];
  /** 创建者，正查给你这个字段是不需要人写的，自动添加 */
  creator?: Maybe<Scalars["String"]["output"]>;
  /** 展示名字， 与metadat.name不一样，这个展示名字是可以用中文的 */
  displayName: Scalars["String"]["output"];
  /** 应用场景 */
  field?: Maybe<Scalars["String"]["output"]>;
  /** 一些用于标记，选择的的标签 */
  labels?: Maybe<Scalars["Map"]["output"]>;
  /** 数据集名称 */
  name: Scalars["String"]["output"];
  /** 数据集所在的namespace，也是后续桶的名字 */
  namespace: Scalars["String"]["output"];
  /** 更新时间, 这里更新指文件同步，或者数据处理完成后，做的更新操作的时间 */
  updateTimestamp?: Maybe<Scalars["Time"]["output"]>;
  /** 数据集的总版本数量 */
  versionCount: Scalars["Int"]["output"];
  /**
   * 这个是一个resolver，数据集下面的版本列表。
   * 支持对名字，类型的完全匹配过滤。
   * 支持通过标签(somelabel=abc)，字段(metadata.name=abc)进行过滤
   */
  versions: PaginatedResult;
};

/**
 * Dataset
 * 数据集代表用户纳管的一组相似属性的文件，采用相同的方式进行数据处理并用于后续的
 * 1. 模型训练
 * 2. 知识库
 *
 * 支持多种类型数据:
 * - 文本
 * - 图片
 * - 视频
 *
 * 单个数据集仅允许包含同一类型文件，不同类型文件将被忽略
 * 数据集允许有多个版本，数据处理针对单个版本进行
 * 数据集某个版本完成数据处理后，数据处理服务需要将处理后的存储回 版本数据集
 */
export type DatasetVersionsArgs = {
  input: ListVersionedDatasetInput;
};

export type DatasetMutation = {
  __typename?: "DatasetMutation";
  createDataset: Dataset;
  /**
   * 删除数据集
   * 可以提供一个名称列表，会将所有名字在这个列表的dataset全部删除
   * 支持通过标签进行删除，提供一个标签选择器，将满足标签的dataset全部删除
   * 如果提供了这两个参数，以名字列表为主。
   */
  deleteDatasets?: Maybe<Scalars["Void"]["output"]>;
  updateDataset: Dataset;
};

export type DatasetMutationCreateDatasetArgs = {
  input?: InputMaybe<CreateDatasetInput>;
};

export type DatasetMutationDeleteDatasetsArgs = {
  input?: InputMaybe<DeleteDatasetInput>;
};

export type DatasetMutationUpdateDatasetArgs = {
  input?: InputMaybe<UpdateDatasetInput>;
};

export type DatasetQuery = {
  __typename?: "DatasetQuery";
  /** 根据名字获取某个具体的数据集 */
  getDataset: Dataset;
  /**
   * 获取数据集列表，支持通过标签和字段进行选择。
   * labelSelector: aa=bbb
   * fieldSelector= metadata.name=somename
   */
  listDatasets: PaginatedResult;
};

export type DatasetQueryGetDatasetArgs = {
  name: Scalars["String"]["input"];
  namespace: Scalars["String"]["input"];
};

export type DatasetQueryListDatasetsArgs = {
  input?: InputMaybe<ListDatasetInput>;
};

export type Datasource = {
  __typename?: "Datasource";
  annotations?: Maybe<Scalars["Map"]["output"]>;
  creator?: Maybe<Scalars["String"]["output"]>;
  description?: Maybe<Scalars["String"]["output"]>;
  displayName: Scalars["String"]["output"];
  endpoint?: Maybe<Endpoint>;
  fileCount?: Maybe<Scalars["Int"]["output"]>;
  labels?: Maybe<Scalars["Map"]["output"]>;
  name: Scalars["String"]["output"];
  namespace: Scalars["String"]["output"];
  oss?: Maybe<Oss>;
  status?: Maybe<Scalars["String"]["output"]>;
  updateTimestamp: Scalars["Time"]["output"];
};

export type DatasourceMutation = {
  __typename?: "DatasourceMutation";
  createDatasource: Datasource;
  deleteDatasource?: Maybe<Scalars["Void"]["output"]>;
  updateDatasource: Datasource;
};

export type DatasourceMutationCreateDatasourceArgs = {
  input: CreateDatasourceInput;
};

export type DatasourceMutationDeleteDatasourceArgs = {
  input?: InputMaybe<DeleteDatasourceInput>;
};

export type DatasourceMutationUpdateDatasourceArgs = {
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

export type DeleteDatasetInput = {
  fieldSelector?: InputMaybe<Scalars["String"]["input"]>;
  labelSelector?: InputMaybe<Scalars["String"]["input"]>;
  name?: InputMaybe<Scalars["String"]["input"]>;
  namespace: Scalars["String"]["input"];
};

export type DeleteDatasourceInput = {
  fieldSelector?: InputMaybe<Scalars["String"]["input"]>;
  /** 筛选器 */
  labelSelector?: InputMaybe<Scalars["String"]["input"]>;
  name?: InputMaybe<Scalars["String"]["input"]>;
  namespace: Scalars["String"]["input"];
};

export type DeleteEmbedderInput = {
  /** 字段选择器 */
  fieldSelector?: InputMaybe<Scalars["String"]["input"]>;
  /** 标签选择器 */
  labelSelector?: InputMaybe<Scalars["String"]["input"]>;
  name?: InputMaybe<Scalars["String"]["input"]>;
  namespace: Scalars["String"]["input"];
};

export type DeleteKnowledgeBaseInput = {
  /** 字段选择器 */
  fieldSelector?: InputMaybe<Scalars["String"]["input"]>;
  /** 标签选择器 */
  labelSelector?: InputMaybe<Scalars["String"]["input"]>;
  name?: InputMaybe<Scalars["String"]["input"]>;
  namespace: Scalars["String"]["input"];
};

export type DeleteModelInput = {
  /** 字段选择器 */
  fieldSelector?: InputMaybe<Scalars["String"]["input"]>;
  /** 标签选择器 */
  labelSelector?: InputMaybe<Scalars["String"]["input"]>;
  name?: InputMaybe<Scalars["String"]["input"]>;
  namespace: Scalars["String"]["input"];
};

export type DeleteVersionedDatasetInput = {
  fieldSelector?: InputMaybe<Scalars["String"]["input"]>;
  labelSelector?: InputMaybe<Scalars["String"]["input"]>;
  name?: InputMaybe<Scalars["String"]["input"]>;
  namespace: Scalars["String"]["input"];
};

export type Embedder = {
  __typename?: "Embedder";
  annotations?: Maybe<Scalars["Map"]["output"]>;
  creator?: Maybe<Scalars["String"]["output"]>;
  description?: Maybe<Scalars["String"]["output"]>;
  displayName: Scalars["String"]["output"];
  endpoint?: Maybe<Endpoint>;
  labels?: Maybe<Scalars["Map"]["output"]>;
  name: Scalars["String"]["output"];
  namespace: Scalars["String"]["output"];
  serviceType?: Maybe<Scalars["String"]["output"]>;
  updateTimestamp?: Maybe<Scalars["Time"]["output"]>;
};

export type EmbedderMutation = {
  __typename?: "EmbedderMutation";
  createEmbedder: Embedder;
  deleteEmbedder?: Maybe<Scalars["Void"]["output"]>;
  updateEmbedder: Embedder;
};

export type EmbedderMutationCreateEmbedderArgs = {
  input: CreateEmbedderInput;
};

export type EmbedderMutationDeleteEmbedderArgs = {
  input?: InputMaybe<DeleteEmbedderInput>;
};

export type EmbedderMutationUpdateEmbedderArgs = {
  input?: InputMaybe<UpdateEmbedderInput>;
};

export type EmbedderQuery = {
  __typename?: "EmbedderQuery";
  getEmbedder: Embedder;
  listEmbedders: PaginatedResult;
};

export type EmbedderQueryGetEmbedderArgs = {
  name: Scalars["String"]["input"];
  namespace: Scalars["String"]["input"];
};

export type EmbedderQueryListEmbeddersArgs = {
  input: ListEmbedderInput;
};

export type Endpoint = {
  __typename?: "Endpoint";
  authSecret?: Maybe<TypedObjectReference>;
  insecure?: Maybe<Scalars["Boolean"]["output"]>;
  url?: Maybe<Scalars["String"]["output"]>;
};

/** 对象存储终端输入 */
export type EndpointInput = {
  /** secret验证密码 */
  authSecret?: InputMaybe<TypedObjectReferenceInput>;
  /** 默认true */
  insecure?: InputMaybe<Scalars["Boolean"]["input"]>;
  url?: InputMaybe<Scalars["String"]["input"]>;
};

/**
 * File
 * 展示某个版本的所有文件。
 */
export type F = {
  __typename?: "F";
  /** 数据量 */
  count?: Maybe<Scalars["Int"]["output"]>;
  /** 文件类型 */
  fileType: Scalars["String"]["output"];
  /** 文件在数据源中的路径，a/b/c.txt或者d.txt */
  path: Scalars["String"]["output"];
  /** 文件成功导入时间，如果没有导入成功，这个字段为空 */
  time?: Maybe<Scalars["Time"]["output"]>;
};

/** 根据条件顾虑版本内的文件，只支持关键词搜索 */
export type FileFilter = {
  /** 根据关键词搜索文件，strings.Container(fileName, keyword) */
  keyword: Scalars["String"]["input"];
  /** 页 */
  page: Scalars["Int"]["input"];
  /** 页内容数量 */
  pageSize: Scalars["Int"]["input"];
  /** 根据文件名字或者更新时间排序, file, time */
  sortBy?: InputMaybe<Scalars["String"]["input"]>;
};

export type FileGroup = {
  /** 用到的文件路径，注意⚠️ 一定不要加bucket的名字 */
  paths?: InputMaybe<Array<Scalars["String"]["input"]>>;
  /** 数据源的基础信息 */
  source: TypedObjectReferenceInput;
};

export type KnowledgeBase = {
  __typename?: "KnowledgeBase";
  annotations?: Maybe<Scalars["Map"]["output"]>;
  creator?: Maybe<Scalars["String"]["output"]>;
  description?: Maybe<Scalars["String"]["output"]>;
  displayName: Scalars["String"]["output"];
  embedder?: Maybe<TypedObjectReference>;
  fileGroups?: Maybe<Array<Maybe<Filegroup>>>;
  labels?: Maybe<Scalars["Map"]["output"]>;
  name: Scalars["String"]["output"];
  namespace: Scalars["String"]["output"];
  /** 知识库连接状态 */
  status?: Maybe<Scalars["String"]["output"]>;
  updateTimestamp: Scalars["Time"]["output"];
  vectorStore?: Maybe<TypedObjectReference>;
};

export type KnowledgeBaseMutation = {
  __typename?: "KnowledgeBaseMutation";
  createKnowledgeBase: KnowledgeBase;
  deleteKnowledgeBase?: Maybe<Scalars["Void"]["output"]>;
  updateKnowledgeBase: KnowledgeBase;
};

export type KnowledgeBaseMutationCreateKnowledgeBaseArgs = {
  input: CreateKnowledgeBaseInput;
};

export type KnowledgeBaseMutationDeleteKnowledgeBaseArgs = {
  input?: InputMaybe<DeleteKnowledgeBaseInput>;
};

export type KnowledgeBaseMutationUpdateKnowledgeBaseArgs = {
  input?: InputMaybe<UpdateKnowledgeBaseInput>;
};

export type KnowledgeBaseQuery = {
  __typename?: "KnowledgeBaseQuery";
  getKnowledgeBase: KnowledgeBase;
  listKnowledgeBases: PaginatedResult;
};

export type KnowledgeBaseQueryGetKnowledgeBaseArgs = {
  name: Scalars["String"]["input"];
  namespace: Scalars["String"]["input"];
};

export type KnowledgeBaseQueryListKnowledgeBasesArgs = {
  input: ListKnowledgeBaseInput;
};

export type ListDatasetInput = {
  displayName?: InputMaybe<Scalars["String"]["input"]>;
  fieldSelector?: InputMaybe<Scalars["String"]["input"]>;
  keyword?: InputMaybe<Scalars["String"]["input"]>;
  labelSelector?: InputMaybe<Scalars["String"]["input"]>;
  name?: InputMaybe<Scalars["String"]["input"]>;
  namespace: Scalars["String"]["input"];
  /** 分页页码，从1开始，默认是1 */
  page?: InputMaybe<Scalars["Int"]["input"]>;
  /** 每页数量，默认10 */
  pageSize?: InputMaybe<Scalars["Int"]["input"]>;
};

/** 分页查询输入 */
export type ListDatasourceInput = {
  /** 数据源资源展示名称 */
  displayName?: InputMaybe<Scalars["String"]["input"]>;
  fieldSelector?: InputMaybe<Scalars["String"]["input"]>;
  keyword?: InputMaybe<Scalars["String"]["input"]>;
  labelSelector?: InputMaybe<Scalars["String"]["input"]>;
  /** 数据源资源名称（不可同名） */
  name?: InputMaybe<Scalars["String"]["input"]>;
  /** 数据源创建命名空间 */
  namespace: Scalars["String"]["input"];
  page?: InputMaybe<Scalars["Int"]["input"]>;
  pageSize?: InputMaybe<Scalars["Int"]["input"]>;
};

export type ListEmbedderInput = {
  displayName?: InputMaybe<Scalars["String"]["input"]>;
  /** 字段选择器 */
  fieldSelector?: InputMaybe<Scalars["String"]["input"]>;
  keyword?: InputMaybe<Scalars["String"]["input"]>;
  /** 标签选择器 */
  labelSelector?: InputMaybe<Scalars["String"]["input"]>;
  name?: InputMaybe<Scalars["String"]["input"]>;
  namespace: Scalars["String"]["input"];
  page?: InputMaybe<Scalars["Int"]["input"]>;
  pageSize?: InputMaybe<Scalars["Int"]["input"]>;
};

export type ListKnowledgeBaseInput = {
  displayName?: InputMaybe<Scalars["String"]["input"]>;
  /** 字段选择器 */
  fieldSelector?: InputMaybe<Scalars["String"]["input"]>;
  keyword?: InputMaybe<Scalars["String"]["input"]>;
  /** 标签选择器 */
  labelSelector?: InputMaybe<Scalars["String"]["input"]>;
  name?: InputMaybe<Scalars["String"]["input"]>;
  namespace: Scalars["String"]["input"];
  page?: InputMaybe<Scalars["Int"]["input"]>;
  pageSize?: InputMaybe<Scalars["Int"]["input"]>;
};

export type ListModelInput = {
  displayName?: InputMaybe<Scalars["String"]["input"]>;
  /** 字段选择器 */
  fieldSelector?: InputMaybe<Scalars["String"]["input"]>;
  keyword?: InputMaybe<Scalars["String"]["input"]>;
  /** 标签选择器 */
  labelSelector?: InputMaybe<Scalars["String"]["input"]>;
  name?: InputMaybe<Scalars["String"]["input"]>;
  namespace: Scalars["String"]["input"];
  page?: InputMaybe<Scalars["Int"]["input"]>;
  pageSize?: InputMaybe<Scalars["Int"]["input"]>;
};

export type ListVersionedDatasetInput = {
  displayName?: InputMaybe<Scalars["String"]["input"]>;
  fieldSelector?: InputMaybe<Scalars["String"]["input"]>;
  keyword?: InputMaybe<Scalars["String"]["input"]>;
  labelSelector?: InputMaybe<Scalars["String"]["input"]>;
  name?: InputMaybe<Scalars["String"]["input"]>;
  namespace: Scalars["String"]["input"];
  /** 分页页码，从1开始，默认是1 */
  page?: InputMaybe<Scalars["Int"]["input"]>;
  /** 每页数量，默认10 */
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
  modeltypes: Scalars["String"]["output"];
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
  Dataset?: Maybe<DatasetMutation>;
  Datasource?: Maybe<DatasourceMutation>;
  Embedder?: Maybe<EmbedderMutation>;
  KnowledgeBase?: Maybe<KnowledgeBaseMutation>;
  Model?: Maybe<ModelMutation>;
  VersionedDataset?: Maybe<VersionedDatasetMutation>;
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

/** 文件输入 */
export type OssInput = {
  Object?: InputMaybe<Scalars["String"]["input"]>;
  bucket?: InputMaybe<Scalars["String"]["input"]>;
};

export type PageNode =
  | Dataset
  | Datasource
  | Embedder
  | F
  | KnowledgeBase
  | Model
  | VersionedDataset;

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
  Dataset?: Maybe<DatasetQuery>;
  Datasource?: Maybe<DatasourceQuery>;
  Embedder?: Maybe<EmbedderQuery>;
  KnowledgeBase?: Maybe<KnowledgeBaseQuery>;
  Model?: Maybe<ModelQuery>;
  VersionedDataset?: Maybe<VersionedDatasetQuery>;
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

export type UpdateDatasetInput = {
  annotations?: InputMaybe<Scalars["Map"]["input"]>;
  /** 同理 */
  description?: InputMaybe<Scalars["String"]["input"]>;
  /** 如果不更新，为空就可以 */
  displayName?: InputMaybe<Scalars["String"]["input"]>;
  /**
   * 更新的的标签信息，这里涉及到增加或者删除标签，
   * 所以，如果标签有任何改动，传递完整的label。
   * 例如之前的标齐是: abc:def 新增一个标签aa:bb, 那么传递 abc:def, aa:bb
   */
  labels?: InputMaybe<Scalars["Map"]["input"]>;
  /** name, namespace用来确定资源，不允许修改的。将原数据传递回来即可。 */
  name: Scalars["String"]["input"];
  namespace: Scalars["String"]["input"];
};

export type UpdateDatasourceInput = {
  /** 数据源资源注释 */
  annotations?: InputMaybe<Scalars["Map"]["input"]>;
  /** 数据源资源描述 */
  description?: InputMaybe<Scalars["String"]["input"]>;
  /** 数据源资源展示名称作为显示，并提供编辑 */
  displayName: Scalars["String"]["input"];
  /** 数据源资源标签 */
  labels?: InputMaybe<Scalars["Map"]["input"]>;
  /** 数据源资源名称（不可同名） */
  name: Scalars["String"]["input"];
  /** 数据源创建命名空间 */
  namespace: Scalars["String"]["input"];
};

export type UpdateEmbedderInput = {
  /** 模型服务资源注释 */
  annotations?: InputMaybe<Scalars["Map"]["input"]>;
  /** 模型服务资源描述 */
  description?: InputMaybe<Scalars["String"]["input"]>;
  /** 模型服务资源展示名称作为显示，并提供编辑 */
  displayName: Scalars["String"]["input"];
  /** 模型服务资源标签 */
  labels?: InputMaybe<Scalars["Map"]["input"]>;
  /** 模型服务资源名称（不可同名） */
  name: Scalars["String"]["input"];
  /** 模型服务创建命名空间 */
  namespace: Scalars["String"]["input"];
};

export type UpdateKnowledgeBaseInput = {
  /** 知识库资源注释 */
  annotations?: InputMaybe<Scalars["Map"]["input"]>;
  /** 知识库资源描述 */
  description?: InputMaybe<Scalars["String"]["input"]>;
  /** 知识库资源展示名称作为显示，并提供编辑 */
  displayName: Scalars["String"]["input"];
  /** 知识库资源标签 */
  labels?: InputMaybe<Scalars["Map"]["input"]>;
  /** 知识库资源名称（不可同名） */
  name: Scalars["String"]["input"];
  /** 知识库创建命名空间 */
  namespace: Scalars["String"]["input"];
};

export type UpdateModelInput = {
  /** 模型资源注释 */
  annotations?: InputMaybe<Scalars["Map"]["input"]>;
  /** 模型资源描述 */
  description?: InputMaybe<Scalars["String"]["input"]>;
  /** 模型资源展示名称作为显示，并提供编辑 */
  displayName: Scalars["String"]["input"];
  /** 模型资标签 */
  labels?: InputMaybe<Scalars["Map"]["input"]>;
  /** 模型资源名称（不可同名） */
  name: Scalars["String"]["input"];
  /** 模型创建命名空间 */
  namespace: Scalars["String"]["input"];
};

export type UpdateVersionedDatasetInput = {
  /** 传递方式同label */
  annotations?: InputMaybe<Scalars["Map"]["input"]>;
  description?: InputMaybe<Scalars["String"]["input"]>;
  displayName: Scalars["String"]["input"];
  /**
   * 更新，删除数据集版本中的文件，传递方式于label相同，完全传递。
   * 如果传递一个空的数组过去，认为是删除全部文件。
   */
  fileGroups?: InputMaybe<Array<FileGroup>>;
  /**
   * 更新的的标签信息，这里涉及到增加或者删除标签，
   * 所以，如果标签有任何改动，传递完整的label。
   * 例如之前的标齐是: abc:def 新增一个标签aa:bb, 那么传递 abc:def, aa:bb
   */
  labels?: InputMaybe<Scalars["Map"]["input"]>;
  /**
   * 这个名字就是metadat.name, 根据name和namespace确定资源
   * name，namespac是不可以更新的。
   */
  name: Scalars["String"]["input"];
  namespace: Scalars["String"]["input"];
};

/**
 * VersionedDataset
 * 数据集的版本信息。
 * 主要记录版本名字，数据的来源，以及文件的同步状态
 */
export type VersionedDataset = {
  __typename?: "VersionedDataset";
  /** 添加一些辅助性记录信息 */
  annotations?: Maybe<Scalars["Map"]["output"]>;
  creationTimestamp: Scalars["Time"]["output"];
  /** 创建者，正查给你这个字段是不需要人写的，自动添加 */
  creator?: Maybe<Scalars["String"]["output"]>;
  /** 数据处理状态，如果为空，表示还没有开始，其他表示 */
  dataProcessStatus?: Maybe<Scalars["String"]["output"]>;
  /** 所属的数据集 */
  dataset: TypedObjectReference;
  /** 展示名字， 与metadat.name不一样，这个展示名字是可以用中文的 */
  displayName: Scalars["String"]["output"];
  /** 该数据集版本所包含的数据总量 */
  fileCount: Scalars["Int"]["output"];
  /** 数据集所包含的文件，对于文件需要支持过滤和分页 */
  files: PaginatedResult;
  /** 一些用于标记，选择的的标签 */
  labels?: Maybe<Scalars["Map"]["output"]>;
  /** 数据集名称, 这个应该是前端随机生成就可以，没有实际用途 */
  name: Scalars["String"]["output"];
  /** 数据集所在的namespace，也是后续桶的名字 */
  namespace: Scalars["String"]["output"];
  /** 该版本是否已经发布, 0是未发布，1是已经发布 */
  released: Scalars["Int"]["output"];
  /** 文件的同步状态, Processing或者'' 表示文件正在同步，Succeede 文件同步成功，Failed 存在文件同步失败 */
  syncStatus?: Maybe<Scalars["String"]["output"]>;
  /** 更新时间, 这里更新指文件同步，或者数据处理完成后，做的更新操作的时间 */
  updateTimestamp?: Maybe<Scalars["Time"]["output"]>;
  /** 版本名称 */
  version: Scalars["String"]["output"];
};

/**
 * VersionedDataset
 * 数据集的版本信息。
 * 主要记录版本名字，数据的来源，以及文件的同步状态
 */
export type VersionedDatasetFilesArgs = {
  input?: InputMaybe<FileFilter>;
};

export type VersionedDatasetMutation = {
  __typename?: "VersionedDatasetMutation";
  createVersionedDataset: VersionedDataset;
  deleteVersionedDatasets?: Maybe<Scalars["Void"]["output"]>;
  updateVersionedDataset: VersionedDataset;
};

export type VersionedDatasetMutationCreateVersionedDatasetArgs = {
  input: CreateVersionedDatasetInput;
};

export type VersionedDatasetMutationDeleteVersionedDatasetsArgs = {
  input: DeleteVersionedDatasetInput;
};

export type VersionedDatasetMutationUpdateVersionedDatasetArgs = {
  input: UpdateVersionedDatasetInput;
};

export type VersionedDatasetQuery = {
  __typename?: "VersionedDatasetQuery";
  getVersionedDataset: VersionedDataset;
  listVersionedDatasets: PaginatedResult;
};

export type VersionedDatasetQueryGetVersionedDatasetArgs = {
  name: Scalars["String"]["input"];
  namespace: Scalars["String"]["input"];
};

export type VersionedDatasetQueryListVersionedDatasetsArgs = {
  input: ListVersionedDatasetInput;
};

export type Filegroup = {
  __typename?: "filegroup";
  path?: Maybe<Array<Scalars["String"]["output"]>>;
  source?: Maybe<TypedObjectReference>;
};

/** 源文件输入 */
export type Filegroupinput = {
  /** 路径 */
  path?: InputMaybe<Array<Scalars["String"]["input"]>>;
  /** 数据源字段 */
  source: TypedObjectReferenceInput;
};

export type ListDatasetsQueryVariables = Exact<{
  input?: InputMaybe<ListDatasetInput>;
  versionsInput: ListVersionedDatasetInput;
  filesInput?: InputMaybe<FileFilter>;
}>;

export type ListDatasetsQuery = {
  __typename?: "Query";
  Dataset?: {
    __typename?: "DatasetQuery";
    listDatasets: {
      __typename?: "PaginatedResult";
      nodes?: Array<
        | {
            __typename?: "Dataset";
            name: string;
            namespace: string;
            creator?: string | null;
            displayName: string;
            updateTimestamp?: any | null;
            contentType: string;
            field?: string | null;
            versionCount: number;
            versions: {
              __typename?: "PaginatedResult";
              nodes?: Array<
                | { __typename?: "Dataset" }
                | { __typename?: "Datasource" }
                | { __typename?: "Embedder" }
                | { __typename?: "F" }
                | { __typename?: "KnowledgeBase" }
                | { __typename?: "Model" }
                | {
                    __typename?: "VersionedDataset";
                    name: string;
                    namespace: string;
                    displayName: string;
                    files: {
                      __typename?: "PaginatedResult";
                      nodes?: Array<
                        | { __typename?: "Dataset" }
                        | { __typename?: "Datasource" }
                        | { __typename?: "Embedder" }
                        | {
                            __typename?: "F";
                            path: string;
                            fileType: string;
                            count?: number | null;
                          }
                        | { __typename?: "KnowledgeBase" }
                        | { __typename?: "Model" }
                        | { __typename?: "VersionedDataset" }
                      > | null;
                    };
                  }
              > | null;
            };
          }
        | { __typename?: "Datasource" }
        | { __typename?: "Embedder" }
        | { __typename?: "F" }
        | { __typename?: "KnowledgeBase" }
        | { __typename?: "Model" }
        | { __typename?: "VersionedDataset" }
      > | null;
    };
  } | null;
};

export type GetDatasetQueryVariables = Exact<{
  name: Scalars["String"]["input"];
  namespace: Scalars["String"]["input"];
}>;

export type GetDatasetQuery = {
  __typename?: "Query";
  Dataset?: {
    __typename?: "DatasetQuery";
    getDataset: {
      __typename?: "Dataset";
      name: string;
      namespace: string;
      creator?: string | null;
      displayName: string;
      updateTimestamp?: any | null;
      contentType: string;
      field?: string | null;
      versionCount: number;
    };
  } | null;
};

export type CreateDatasetMutationVariables = Exact<{
  input?: InputMaybe<CreateDatasetInput>;
}>;

export type CreateDatasetMutation = {
  __typename?: "Mutation";
  Dataset?: {
    __typename?: "DatasetMutation";
    createDataset: {
      __typename?: "Dataset";
      name: string;
      displayName: string;
      labels?: any | null;
    };
  } | null;
};

export type UpdateDatasetMutationVariables = Exact<{
  input?: InputMaybe<UpdateDatasetInput>;
}>;

export type UpdateDatasetMutation = {
  __typename?: "Mutation";
  Dataset?: {
    __typename?: "DatasetMutation";
    updateDataset: {
      __typename?: "Dataset";
      name: string;
      displayName: string;
      labels?: any | null;
    };
  } | null;
};

export type DeleteDatasetsMutationVariables = Exact<{
  input?: InputMaybe<DeleteDatasetInput>;
}>;

export type DeleteDatasetsMutation = {
  __typename?: "Mutation";
  Dataset?: {
    __typename?: "DatasetMutation";
    deleteDatasets?: any | null;
  } | null;
};

export type CreateDatasourceMutationVariables = Exact<{
  input: CreateDatasourceInput;
}>;

export type CreateDatasourceMutation = {
  __typename?: "Mutation";
  Datasource?: {
    __typename?: "DatasourceMutation";
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
    __typename?: "DatasourceMutation";
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
    __typename?: "DatasourceMutation";
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
        | { __typename: "Dataset" }
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
        | { __typename: "Embedder" }
        | { __typename: "F" }
        | { __typename: "KnowledgeBase" }
        | { __typename: "Model" }
        | { __typename: "VersionedDataset" }
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
        | { __typename: "Dataset" }
        | { __typename: "Datasource" }
        | { __typename: "Embedder" }
        | { __typename: "F" }
        | { __typename: "KnowledgeBase" }
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
            modeltypes: string;
            updateTimestamp?: any | null;
          }
        | { __typename: "VersionedDataset" }
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
      modeltypes: string;
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
      modeltypes: string;
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
      modeltypes: string;
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

export type CreateVersionedDatasetMutationVariables = Exact<{
  input: CreateVersionedDatasetInput;
}>;

export type CreateVersionedDatasetMutation = {
  __typename?: "Mutation";
  VersionedDataset?: {
    __typename?: "VersionedDatasetMutation";
    createVersionedDataset: {
      __typename?: "VersionedDataset";
      name: string;
      displayName: string;
      creator?: string | null;
      namespace: string;
      version: string;
      updateTimestamp?: any | null;
      creationTimestamp: any;
      fileCount: number;
      released: number;
      syncStatus?: string | null;
      dataProcessStatus?: string | null;
    };
  } | null;
};

export type UpdateVersionedDatasetMutationVariables = Exact<{
  input: UpdateVersionedDatasetInput;
}>;

export type UpdateVersionedDatasetMutation = {
  __typename?: "Mutation";
  VersionedDataset?: {
    __typename?: "VersionedDatasetMutation";
    updateVersionedDataset: {
      __typename?: "VersionedDataset";
      name: string;
      displayName: string;
    };
  } | null;
};

export type DeleteVersionedDatasetsMutationVariables = Exact<{
  input: DeleteVersionedDatasetInput;
}>;

export type DeleteVersionedDatasetsMutation = {
  __typename?: "Mutation";
  VersionedDataset?: {
    __typename?: "VersionedDatasetMutation";
    deleteVersionedDatasets?: any | null;
  } | null;
};

export type GetVersionedDatasetQueryVariables = Exact<{
  name: Scalars["String"]["input"];
  namespace: Scalars["String"]["input"];
  fileInput?: InputMaybe<FileFilter>;
}>;

export type GetVersionedDatasetQuery = {
  __typename?: "Query";
  VersionedDataset?: {
    __typename?: "VersionedDatasetQuery";
    getVersionedDataset: {
      __typename?: "VersionedDataset";
      name: string;
      displayName: string;
      namespace: string;
      creator?: string | null;
      files: {
        __typename?: "PaginatedResult";
        nodes?: Array<
          | { __typename?: "Dataset" }
          | { __typename?: "Datasource" }
          | { __typename?: "Embedder" }
          | {
              __typename?: "F";
              path: string;
              time?: any | null;
              fileType: string;
              count?: number | null;
            }
          | { __typename?: "KnowledgeBase" }
          | { __typename?: "Model" }
          | { __typename?: "VersionedDataset" }
        > | null;
      };
    };
  } | null;
};

export type ListVersionedDatasetsQueryVariables = Exact<{
  input: ListVersionedDatasetInput;
  fileInput?: InputMaybe<FileFilter>;
}>;

export type ListVersionedDatasetsQuery = {
  __typename?: "Query";
  VersionedDataset?: {
    __typename?: "VersionedDatasetQuery";
    listVersionedDatasets: {
      __typename?: "PaginatedResult";
      nodes?: Array<
        | { __typename?: "Dataset" }
        | { __typename?: "Datasource" }
        | { __typename?: "Embedder" }
        | { __typename?: "F" }
        | { __typename?: "KnowledgeBase" }
        | { __typename?: "Model" }
        | {
            __typename?: "VersionedDataset";
            name: string;
            displayName: string;
            namespace: string;
            creator?: string | null;
            files: {
              __typename?: "PaginatedResult";
              nodes?: Array<
                | { __typename?: "Dataset" }
                | { __typename?: "Datasource" }
                | { __typename?: "Embedder" }
                | {
                    __typename?: "F";
                    path: string;
                    time?: any | null;
                    fileType: string;
                    count?: number | null;
                  }
                | { __typename?: "KnowledgeBase" }
                | { __typename?: "Model" }
                | { __typename?: "VersionedDataset" }
              > | null;
            };
          }
      > | null;
    };
  } | null;
};

export const ListDatasetsDocument = gql`
  query listDatasets(
    $input: ListDatasetInput
    $versionsInput: ListVersionedDatasetInput!
    $filesInput: FileFilter
  ) {
    Dataset {
      listDatasets(input: $input) {
        nodes {
          ... on Dataset {
            name
            namespace
            creator
            displayName
            updateTimestamp
            contentType
            field
            versionCount
            versions(input: $versionsInput) {
              nodes {
                ... on VersionedDataset {
                  name
                  namespace
                  displayName
                  files(input: $filesInput) {
                    nodes {
                      ... on F {
                        path
                        fileType
                        count
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }
    }
  }
`;
export const GetDatasetDocument = gql`
  query getDataset($name: String!, $namespace: String!) {
    Dataset {
      getDataset(name: $name, namespace: $namespace) {
        name
        namespace
        creator
        displayName
        updateTimestamp
        contentType
        field
        versionCount
      }
    }
  }
`;
export const CreateDatasetDocument = gql`
  mutation createDataset($input: CreateDatasetInput) {
    Dataset {
      createDataset(input: $input) {
        name
        displayName
        labels
      }
    }
  }
`;
export const UpdateDatasetDocument = gql`
  mutation updateDataset($input: UpdateDatasetInput) {
    Dataset {
      updateDataset(input: $input) {
        name
        displayName
        labels
      }
    }
  }
`;
export const DeleteDatasetsDocument = gql`
  mutation deleteDatasets($input: DeleteDatasetInput) {
    Dataset {
      deleteDatasets(input: $input)
    }
  }
`;
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
            modeltypes
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
        modeltypes
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
        modeltypes
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
        modeltypes
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
export const CreateVersionedDatasetDocument = gql`
  mutation createVersionedDataset($input: CreateVersionedDatasetInput!) {
    VersionedDataset {
      createVersionedDataset(input: $input) {
        name
        displayName
        creator
        namespace
        version
        updateTimestamp
        creationTimestamp
        fileCount
        released
        syncStatus
        dataProcessStatus
      }
    }
  }
`;
export const UpdateVersionedDatasetDocument = gql`
  mutation updateVersionedDataset($input: UpdateVersionedDatasetInput!) {
    VersionedDataset {
      updateVersionedDataset(input: $input) {
        name
        displayName
      }
    }
  }
`;
export const DeleteVersionedDatasetsDocument = gql`
  mutation deleteVersionedDatasets($input: DeleteVersionedDatasetInput!) {
    VersionedDataset {
      deleteVersionedDatasets(input: $input)
    }
  }
`;
export const GetVersionedDatasetDocument = gql`
  query getVersionedDataset(
    $name: String!
    $namespace: String!
    $fileInput: FileFilter
  ) {
    VersionedDataset {
      getVersionedDataset(name: $name, namespace: $namespace) {
        name
        displayName
        namespace
        creator
        files(input: $fileInput) {
          nodes {
            ... on F {
              path
              time
              fileType
              count
            }
          }
        }
      }
    }
  }
`;
export const ListVersionedDatasetsDocument = gql`
  query listVersionedDatasets(
    $input: ListVersionedDatasetInput!
    $fileInput: FileFilter
  ) {
    VersionedDataset {
      listVersionedDatasets(input: $input) {
        nodes {
          ... on VersionedDataset {
            name
            displayName
            namespace
            creator
            files(input: $fileInput) {
              nodes {
                ... on F {
                  path
                  time
                  fileType
                  count
                }
              }
            }
          }
        }
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
    listDatasets(
      variables: ListDatasetsQueryVariables,
      requestHeaders?: GraphQLClientRequestHeaders,
    ): Promise<ListDatasetsQuery> {
      return withWrapper(
        (wrappedRequestHeaders) =>
          client.request<ListDatasetsQuery>(ListDatasetsDocument, variables, {
            ...requestHeaders,
            ...wrappedRequestHeaders,
          }),
        "listDatasets",
        "query",
      );
    },
    getDataset(
      variables: GetDatasetQueryVariables,
      requestHeaders?: GraphQLClientRequestHeaders,
    ): Promise<GetDatasetQuery> {
      return withWrapper(
        (wrappedRequestHeaders) =>
          client.request<GetDatasetQuery>(GetDatasetDocument, variables, {
            ...requestHeaders,
            ...wrappedRequestHeaders,
          }),
        "getDataset",
        "query",
      );
    },
    createDataset(
      variables?: CreateDatasetMutationVariables,
      requestHeaders?: GraphQLClientRequestHeaders,
    ): Promise<CreateDatasetMutation> {
      return withWrapper(
        (wrappedRequestHeaders) =>
          client.request<CreateDatasetMutation>(
            CreateDatasetDocument,
            variables,
            { ...requestHeaders, ...wrappedRequestHeaders },
          ),
        "createDataset",
        "mutation",
      );
    },
    updateDataset(
      variables?: UpdateDatasetMutationVariables,
      requestHeaders?: GraphQLClientRequestHeaders,
    ): Promise<UpdateDatasetMutation> {
      return withWrapper(
        (wrappedRequestHeaders) =>
          client.request<UpdateDatasetMutation>(
            UpdateDatasetDocument,
            variables,
            { ...requestHeaders, ...wrappedRequestHeaders },
          ),
        "updateDataset",
        "mutation",
      );
    },
    deleteDatasets(
      variables?: DeleteDatasetsMutationVariables,
      requestHeaders?: GraphQLClientRequestHeaders,
    ): Promise<DeleteDatasetsMutation> {
      return withWrapper(
        (wrappedRequestHeaders) =>
          client.request<DeleteDatasetsMutation>(
            DeleteDatasetsDocument,
            variables,
            { ...requestHeaders, ...wrappedRequestHeaders },
          ),
        "deleteDatasets",
        "mutation",
      );
    },
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
    createVersionedDataset(
      variables: CreateVersionedDatasetMutationVariables,
      requestHeaders?: GraphQLClientRequestHeaders,
    ): Promise<CreateVersionedDatasetMutation> {
      return withWrapper(
        (wrappedRequestHeaders) =>
          client.request<CreateVersionedDatasetMutation>(
            CreateVersionedDatasetDocument,
            variables,
            { ...requestHeaders, ...wrappedRequestHeaders },
          ),
        "createVersionedDataset",
        "mutation",
      );
    },
    updateVersionedDataset(
      variables: UpdateVersionedDatasetMutationVariables,
      requestHeaders?: GraphQLClientRequestHeaders,
    ): Promise<UpdateVersionedDatasetMutation> {
      return withWrapper(
        (wrappedRequestHeaders) =>
          client.request<UpdateVersionedDatasetMutation>(
            UpdateVersionedDatasetDocument,
            variables,
            { ...requestHeaders, ...wrappedRequestHeaders },
          ),
        "updateVersionedDataset",
        "mutation",
      );
    },
    deleteVersionedDatasets(
      variables: DeleteVersionedDatasetsMutationVariables,
      requestHeaders?: GraphQLClientRequestHeaders,
    ): Promise<DeleteVersionedDatasetsMutation> {
      return withWrapper(
        (wrappedRequestHeaders) =>
          client.request<DeleteVersionedDatasetsMutation>(
            DeleteVersionedDatasetsDocument,
            variables,
            { ...requestHeaders, ...wrappedRequestHeaders },
          ),
        "deleteVersionedDatasets",
        "mutation",
      );
    },
    getVersionedDataset(
      variables: GetVersionedDatasetQueryVariables,
      requestHeaders?: GraphQLClientRequestHeaders,
    ): Promise<GetVersionedDatasetQuery> {
      return withWrapper(
        (wrappedRequestHeaders) =>
          client.request<GetVersionedDatasetQuery>(
            GetVersionedDatasetDocument,
            variables,
            { ...requestHeaders, ...wrappedRequestHeaders },
          ),
        "getVersionedDataset",
        "query",
      );
    },
    listVersionedDatasets(
      variables: ListVersionedDatasetsQueryVariables,
      requestHeaders?: GraphQLClientRequestHeaders,
    ): Promise<ListVersionedDatasetsQuery> {
      return withWrapper(
        (wrappedRequestHeaders) =>
          client.request<ListVersionedDatasetsQuery>(
            ListVersionedDatasetsDocument,
            variables,
            { ...requestHeaders, ...wrappedRequestHeaders },
          ),
        "listVersionedDatasets",
        "query",
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
    useListDatasets(
      variables: ListDatasetsQueryVariables,
      config?: SWRConfigInterface<ListDatasetsQuery, ClientError>,
    ) {
      return useSWR<ListDatasetsQuery, ClientError>(
        genKey<ListDatasetsQueryVariables>("ListDatasets", variables),
        () => sdk.listDatasets(variables),
        config,
      );
    },
    useGetDataset(
      variables: GetDatasetQueryVariables,
      config?: SWRConfigInterface<GetDatasetQuery, ClientError>,
    ) {
      return useSWR<GetDatasetQuery, ClientError>(
        genKey<GetDatasetQueryVariables>("GetDataset", variables),
        () => sdk.getDataset(variables),
        config,
      );
    },
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
    useGetVersionedDataset(
      variables: GetVersionedDatasetQueryVariables,
      config?: SWRConfigInterface<GetVersionedDatasetQuery, ClientError>,
    ) {
      return useSWR<GetVersionedDatasetQuery, ClientError>(
        genKey<GetVersionedDatasetQueryVariables>(
          "GetVersionedDataset",
          variables,
        ),
        () => sdk.getVersionedDataset(variables),
        config,
      );
    },
    useListVersionedDatasets(
      variables: ListVersionedDatasetsQueryVariables,
      config?: SWRConfigInterface<ListVersionedDatasetsQuery, ClientError>,
    ) {
      return useSWR<ListVersionedDatasetsQuery, ClientError>(
        genKey<ListVersionedDatasetsQueryVariables>(
          "ListVersionedDatasets",
          variables,
        ),
        () => sdk.listVersionedDatasets(variables),
        config,
      );
    },
  };
}
export type SdkWithHooks = ReturnType<typeof getSdkWithHooks>;

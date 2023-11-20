import { GraphQLClient } from "graphql-request";
import type {
  RequestConfig,
  RequestMiddleware,
  Response,
} from "graphql-request/src/types";
import type { RequestInit } from "graphql-request/src/types.dom";
import qs from "query-string";
import { useMemo } from "react";
import { errorsHandler } from "./errors";
import { getSdk, getSdkWithHooks } from "./sdk";

export * from "graphql-request";
export * from "./errors";
export * from "./sdk";

const isProd = process.env.NODE_ENV === "production";
const AUTH_DATA = "authData";
const getAuthData = () => {
  try {
    const authData = JSON.parse(window.localStorage.getItem(AUTH_DATA) || "{}");
    return authData;
  } catch (error) {
    console.warn("getAuthData failed", error);
    return {};
  }
};

export interface RequestOptions extends RequestInit {
  url: string;
}

export const requestMiddleware: RequestMiddleware<any> = (request) => {
  const { url, operationName, ...otherRequest } = request;
  const [host, search] = url.split("?");
  const query = qs.parse(search);
  // query 中增加操作名，便于定位问题
  if (operationName) {
    query.o = operationName;
  }
  const authData = getAuthData();
  const { token_type, id_token } = authData.token || {};
  const Authorization = token_type && id_token && `${token_type} ${id_token}`;
  return {
    url: `${host}?${qs.stringify(query)}`,
    operationName,
    credentials: "include",
    ...otherRequest,
    headers: Authorization
      ? { ...request.headers, Authorization }
      : { ...request.headers },
  };
};

export const responseMiddleware = (response: Response<unknown> | Error) => {
  const errors: Error[] = response.errors || response.response?.errors;
  if (errors) {
    errorsHandler(errors);
  }
};

const devEndpoint = "/kubeagi-apis/bff";
const prodEndpoint = "/kubeagi-apis/bff";
const endpoint = isProd ? prodEndpoint : devEndpoint;

export const client = new GraphQLClient(endpoint, {
  requestMiddleware,
  responseMiddleware,
});
export const sdk = getSdkWithHooks(client);

/** 初始化 sdk 的配置项 */
interface SdkBaseOptions {
  /** api 地址，默认为 `'/bff'` */
  url?: string;
  /** 是否返回 hooks 相关函数，默认为 `false` */
  withHooks?: boolean;
  /** 请求配置项 */
  requestConfig?: RequestConfig;
}

export type SdkOptions = Pick<SdkBaseOptions, "url" | "requestConfig">;

/**
 * 初始化 GraphQL client 实例
 *
 * @param {string} url api 地址，默认为 `'/bff'`
 * @param {RequestConfig} [requestConfig] 请求配置项
 * @return {GraphQLClient} GraphQL client 实例
 */
export const initGraphQLClient = (
  url?: string,
  requestConfig?: RequestConfig,
) => {
  if (!requestConfig) {
    requestConfig = {};
  }
  if (!requestConfig.requestMiddleware) {
    requestConfig.requestMiddleware = requestMiddleware;
  } else {
    requestConfig.requestMiddleware = async (request) => {
      const newReq = await requestConfig!.requestMiddleware!(request);
      return requestMiddleware(newReq);
    };
  }
  if (!requestConfig.responseMiddleware) {
    requestConfig.responseMiddleware = responseMiddleware;
  }
  return new GraphQLClient(url || endpoint, requestConfig);
};

const initSdkBase = (options: SdkBaseOptions = {}) => {
  const { url, withHooks, requestConfig } = options;

  const _client = initGraphQLClient(url, requestConfig);
  const _sdk = withHooks ? getSdkWithHooks(_client) : getSdk(_client);
  return _sdk;
};

/**
 * 初始化 sdk 实例
 *
 * @param {SdkOptions} options 配置项
 */
export const initSdk = (options: SdkOptions = {}) => {
  const { url, requestConfig } = options;
  const _sdk = initSdkBase({ withHooks: false, url, requestConfig });
  return _sdk;
};

/**
 * 初始化 sdk 实例 (包含 hooks)
 *
 * @param {SdkOptions} options 配置项
 */
export const initSdkWithHooks = (options: SdkOptions = {}) => {
  const { url, requestConfig } = options;
  const _sdk = initSdkBase({ withHooks: true, url, requestConfig });
  return _sdk as ReturnType<typeof getSdkWithHooks>;
};

/**
 * hook 的方式获取 sdk 实例
 *
 * @param {SdkOptions} options 配置项
 */
export const useSdk = (options: SdkOptions = {}) => {
  const { url, requestConfig } = options;
  return useMemo(
    () => initSdkWithHooks({ url, requestConfig }),
    [url, requestConfig],
  );
};

import { ClientError } from "graphql-request";
import type { Fetcher, Key, SWRConfiguration, SWRResponse } from "swr";
import __useSWR from "swr";

export interface SWRResponsePro<Data = any, Error = any>
  extends SWRResponse<Data, Error> {
  loading: boolean;
}

const SWR_ONFOCUS = "swr_revalidateOnFocus"; // 聚焦时重新请求
const SWR_IFSTALE = "swr_revalidateIfStale"; // 控制SWR在挂载并且存在陈旧数据时重新请求
const SWR_ONRECONNECT = "swr_revalidateOnReconnect"; // 重新连接时重新请求
const defaultConfig = {
  revalidateOnFocus: window.localStorage.getItem(SWR_ONFOCUS) !== "false",
  revalidateIfStale: window.localStorage.getItem(SWR_IFSTALE) !== "false",
  revalidateOnReconnect:
    window.localStorage.getItem(SWR_ONRECONNECT) !== "false",
};

export const useSWR = <
  Data = any,
  Error = ClientError,
  SWRKey extends Key = null,
>(
  key: SWRKey,
  fetcher: Fetcher<Data, SWRKey> | null,
  config: SWRConfiguration<Data, Error, Fetcher<Data, SWRKey>> | undefined,
): SWRResponsePro<Data, Error> => {
  const res = __useSWR<Data, Error, SWRKey>(key, fetcher, {
    ...defaultConfig,
    ...(config || {}),
  });
  const { error, data } = res;

  const hasObjectData =
    data &&
    Object.prototype.toString.call(data) === "[object Object]" &&
    Object.keys(data || {}).length > 0;
  return {
    ...res,
    data: hasObjectData ? data : error?.response?.data || data,
    loading: !data && !error,
  };
};

export const genKey = <
  V extends Record<string, unknown> = Record<string, unknown>,
>(
  name: string,
  object: V = {} as V,
): Key => [
  name,
  ...Object.keys(object)
    .sort()
    .map((key) => object[key]),
];

export default useSWR;

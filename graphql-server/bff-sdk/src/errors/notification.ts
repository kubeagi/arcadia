import { notification } from "@tenx-ui/materials";
import { GraphQLError } from "graphql-request/src/types";

const VERBS_MAP = {
  create: "创建",
  delete: "删除",
  update: "更新",
  patch: "更新",
  get: "获取",
  list: "列取",
  watch: "监听",
};
export const showForbiddenNotification = (error: GraphQLError) => {
  const { name, kind, verb } = error.extensions?.exception?.details || {};
  let description = "当前用户没有权限";
  description += `${VERBS_MAP[verb] || "操作"}`;
  if (kind) {
    description += ` ${kind}`;
  }
  if (name) {
    description += ` ${name}`;
  }
  notification.warn({
    message: "当前操作未被授权",
    description,
    detail: error,
  });
};

export const showGlobalErrorNotification = (error: GraphQLError) => {
  const { message } = error || {};

  notification.warn({
    message: "请求错误",
    description: message,
    detail: error,
  });
};

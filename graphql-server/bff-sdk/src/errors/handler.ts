import { GraphQLError } from "graphql-request/src/types";
import { showInvalidTokenModal } from "./modal";
import {
  showForbiddenNotification,
  showGlobalErrorNotification,
} from "./notification";

export const errorsHandler = (errors: GraphQLError[]) => {
  const gqlErrors = errors.filter(
    (e) => typeof e.extensions?.code !== "undefined",
  );
  if (gqlErrors.length === 0) {
    console.warn("uncaught errors", errors);
    return;
  }
  gqlErrors.forEach((e) => {
    switch (e.extensions.code) {
      case "InvalidToken":
        showInvalidTokenModal(e);
        break;
      case "Forbidden":
        showForbiddenNotification(e);
        break;
      default:
        // showGlobalErrorNotification(e);
        break;
    }
  });
};

export default errorsHandler;

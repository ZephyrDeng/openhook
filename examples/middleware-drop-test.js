if (ctx.environment === "test") {
  return { reject: true, message: "test environment ignored" };
}

headers["X-OpenHook-Source"] = "custom-middleware";
ctx.severity = ctx.severity || "warning";
return true;

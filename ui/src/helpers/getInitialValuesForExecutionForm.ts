import { get } from "lodash"
import getOwnerIdRunTagFromCookies from "./getOwnerIdRunTagFromCookies"
import {
  Executable,
  LaunchRequestV2,
  Run,
  Task,
  Template,
  TemplateExecutionRequest,
  ExecutionRequestCommon,
  ExecutionEngine,
  Env,
  DefaultNodeLifecycle,
  DefaultExecutionEngine,
} from "../types"
import constructDefaultObjectFromJsonSchema from "./constructDefaultObjectFromJsonSchema"

export function getInitialValuesForTemplateExecutionForm(
  t: Template,
  r: Run | null
): TemplateExecutionRequest {
  const req: TemplateExecutionRequest = {
    ...getInitialValuesForCommonExecutionFields(t, r),
    template_payload: get(
      r,
      ["execution_request_custom", "template_payload"],
      constructDefaultObjectFromJsonSchema(t.schema)
    ),
  }

  return req
}

function getInitialValuesForCommonExecutionFields(
  e: Executable,
  r: Run | null
): ExecutionRequestCommon {
  // Set ownerID value.
  const ownerID = get(
    r,
    ["run_tags", "owner_id"],
    getOwnerIdRunTagFromCookies()
  )

  // Set env value.
  let env: Env[] | null = r && r.env ? r.env : e.env

  // Filter out invalid run env if specified in dotenv file.
  if (env === null) {
    env = []
  } else if (process.env.REACT_APP_INVALID_RUN_ENV !== undefined) {
    const invalidEnvs = new Set(
      process.env.REACT_APP_INVALID_RUN_ENV.split(",")
    )
    env = env.filter(e => !invalidEnvs.has(e.name))
  }

  // Set CPU value.
  let cpu: number = r && r.cpu ? r.cpu : e.cpu
  if (cpu < 512) cpu = 512

  // Set memory value.
  const memory: number = r && r.memory ? r.memory : e.memory

  // Set engine.
  const engine: ExecutionEngine = get(r, "engine", DefaultExecutionEngine)


      return {
        env,
        cpu,
        memory,
        owner_id: ownerID,
        engine,
      }
}

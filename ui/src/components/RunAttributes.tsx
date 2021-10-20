import * as React from "react"
import { Card, Pre, Tag } from "@blueprintjs/core"
import { Run } from "../types"
import Attribute from "./Attribute"
import ISO8601AttributeValue from "./ISO8601AttributeValue"

const RunAttributes: React.FC<{ data: Run }> = ({ data }) => (
  <Card style={{ marginBottom: 12 }}>
    <div
      className="flotilla-attributes-container flotilla-attributes-container-horizontal"
      style={{ marginBottom: 12 }}
    >
      <Attribute name="Engine Type" value={<Tag>{data.engine}</Tag>} />
    </div>
    <div className="flotilla-form-section-divider" />

      <div className="flotilla-attributes-container flotilla-attributes-container-horizontal">
        {data.max_cpu_used && <Attribute
          name="CPU Used"
          value={`${data.max_cpu_used} units`}
        />}
        <Attribute
          name="CPU Requested"
          value={data.cpu ? `${data.cpu} units` : ""}
        />
        <Attribute
          name="CPU Limit"
          value={data.cpu_limit ? `${data.cpu_limit} units` : ""}
        />
      </div>

      <div className="flotilla-form-section-divider" />

      <div className="flotilla-attributes-container flotilla-attributes-container-horizontal">
        {data.max_memory_used && <Attribute
          name="Memory Used"
          value={`${data.max_memory_used} MB`}
        />}
        <Attribute
          name="Memory Requested"
          value={`${data.memory} MB`}
        />
        <Attribute
          name="Memory Limit"
          value={`${data.memory_limit} MB`}
        />
      </div>




    {data.gpu && (
      <div
        className="flotilla-attributes-container flotilla-attributes-container-horizontal"
        style={{ marginBottom: 12 }}
      >
        <Attribute name="GPU Count" value={data.gpu || 0} />
      </div>
    )}
    <div className="flotilla-form-section-divider" />
    <div
      className="flotilla-attributes-container flotilla-attributes-container-horizontal"
      style={{ marginBottom: 12 }}
    >
      <Attribute
        name="Queued At"
        value={<ISO8601AttributeValue time={data.queued_at} />}
      />
      <Attribute
        name="Started At"
        value={<ISO8601AttributeValue time={data.started_at} />}
      />
      <Attribute
        name="Finished At"
        value={<ISO8601AttributeValue time={data.finished_at} />}
      />
    </div>
    <div className="flotilla-form-section-divider" />
    <div className="flotilla-attributes-container flotilla-attributes-container-vertical">
      <Attribute
        name="Run ID"
        value={data.run_id}
        isCopyable
        rawValue={data.run_id}
      />
      <Attribute
        name="Template ID"
        value={data.template_id}
        isCopyable
        rawValue={data.template_id}
      />
      <Attribute name="Image" value={data.image} />
      <Attribute
        name="Command"
        value={
          data.command ? (
            <Pre className="flotilla-pre">
              {data.command.replace(/\n(\s)+/g, "\n")}
            </Pre>
          ) : (
              "Existing task definition command was used."
            )
        }
      />
    </div>
  </Card>
)

export default RunAttributes

import * as React from "react"
import { FieldArray, FastField, FormikErrors } from "formik"
import { get } from "lodash"
import { Button, FormGroup, Classes, Intent } from "@blueprintjs/core"
import { Env } from "../types"
import { IconNames } from "@blueprintjs/icons"
import { envFieldSpec } from "../helpers/taskFormHelpers"
import FieldError from "./FieldError"

export type Props = {
  values: Env[]
  push: (env: Env) => void
  remove: (index: number) => void
  errors: string | FormikErrors<any> | string[] | FormikErrors<any>[] | undefined
}

export const EnvFieldArray: React.FunctionComponent<Props> = ({
  values,
  push,
  remove,
  errors,
}) => (
  <div>
    <div className="flotilla-form-section-header-container">
      <div>{envFieldSpec.label}</div>
      <Button
        onClick={() => {
          push({ name: "", value: "" })
        }}
        type="button"
        className="flotilla-env-field-array-add-button"
      >
        Add
      </Button>
    </div>
    <div>
      {values.map((env: Env, i: number) => (
        <div key={i} className="flotilla-env-field-array-item">
          <FormGroup label={i === 0 ? "Name" : null}>
            <FastField
              name={`${envFieldSpec.name}[${i}].name`}
              className={Classes.INPUT}
            />
            <FieldError>{get(errors, [i, "name"], null)}</FieldError>
          </FormGroup>
          <FormGroup label={i === 0 ? "Value" : null}>
            <FastField
              name={`${envFieldSpec.name}[${i}].value`}
              className={Classes.INPUT}
            />
            <FieldError>{get(errors, [i, "value"], null)}</FieldError>
          </FormGroup>
          <Button
            onClick={() => {
              remove(i)
            }}
            type="button"
            intent={Intent.DANGER}
            style={i === 0 ? { transform: `translateY(8px)` } : {}}
            icon={IconNames.CROSS}
          ></Button>
        </div>
      ))}
    </div>
  </div>
)

const ConnectedEnvFieldArray: React.FunctionComponent<{}> = () => (
  <FieldArray name={envFieldSpec.name}>
    {({ form, push, remove }) => (
      <EnvFieldArray
        values={form.values.env}
        push={push}
        remove={remove}
        errors={form.errors.env}
      />
    )}
  </FieldArray>
)

export default ConnectedEnvFieldArray

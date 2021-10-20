import * as React from "react"
import {FastField, Form, Formik} from "formik"
import * as Yup from "yup"
import {RouteComponentProps} from "react-router-dom"
import {Button, Classes, FormGroup, Intent, Radio, RadioGroup, Spinner,} from "@blueprintjs/core"
import api from "../api"
import {ExecutionEngine, Run, Template, TemplateExecutionRequest,} from "../types"
import Request, {ChildProps as RequestChildProps, RequestStatus,} from "./Request"
import EnvFieldArray from "./EnvFieldArray"
import {TemplateContext, TemplateCtx} from "./Template"
import Toaster from "./Toaster"
import ErrorCallout from "./ErrorCallout"
import FieldError from "./FieldError"
import * as helpers from "../helpers/runFormHelpers"
import JSONSchemaForm, {ArrayFieldTemplateProps, FieldTemplateProps,} from "react-jsonschema-form"

const getInitialValuesForTemplateRun = (): TemplateExecutionRequest => {
  return {
    template_payload: {},
    env: [],
    owner_id: "",
    memory: 512,
    cpu: 512,
    engine: ExecutionEngine.LOCAL,
  }
}

const validationSchema = Yup.object().shape({
  owner_id: Yup.string(),
  cluster: Yup.string().required("Required"),
  memory: Yup.number()
    .required("Required")
    .min(0),
  cpu: Yup.number()
    .required("Required")
    .min(512),
  env: Yup.array().of(
    Yup.object().shape({
      name: Yup.string().required(),
      value: Yup.string().required(),
    })
  ),
  engine: Yup.string()
    .matches(/(eks|ecs|local)/)
    .required("A valid engine type of ecs, eks, or local must be set."),
  template_payload: Yup.object().required("template_payload is required"),
})

type Props = RequestChildProps<
  Run,
  { templateID: string; data: TemplateExecutionRequest }
> & {
  templateID: string
  initialValues: TemplateExecutionRequest
  template: Template
}

const FieldTemplate: React.FC<FieldTemplateProps> = props => {
  return (
    <FormGroup
      label={props.label}
      helperText={props.description}
      labelInfo={props.required ? "(Required)" : ""}
    >
      {props.children}
    </FormGroup>
  )
}

const ArrayFieldTemplate: React.FC<ArrayFieldTemplateProps> = props => {
  return (
    <div>
      {props.items.map((element, i) =>
        React.cloneElement(element.children, { key: i })
      )}
      {props.canAdd && (
        <Button type="button" onClick={props.onAddClick} icon="plus" fill>
          Add {props.title}
        </Button>
      )}
    </div>
  )
}

class RunForm extends React.Component<Props> {
  private FORMIK_REF = React.createRef<Formik<TemplateExecutionRequest>>()

  // Note: this method is a bit hacky as we have two form elements - Formik (F)
  // and JSONSchemaForm (J). F does not have a submit button, J does. When J's
  // submit button is clicked, this method is called. We get the values of the
  // F form via the `FORMIK_REF` ref binding. Then we take the J form's values
  // and shove them into F form's `template_payload` field. This request is
  // then sent to the server.
  onSubmit(jsonschemaForm: any) {
    if (this.FORMIK_REF.current) {
      const formikValues = this.FORMIK_REF.current.state.values
      formikValues["template_payload"] = jsonschemaForm
      this.props.request({
        templateID: this.props.templateID,
        data: formikValues,
      })
    }
  }

  render() {
    const {
      initialValues,
      request,
      requestStatus,
      isLoading,
      error,
      templateID,
      template,
    } = this.props

    return (
      <div className="flotilla-form-container">
        <Formik<TemplateExecutionRequest>
          ref={this.FORMIK_REF}
          isInitialValid={(values: any) =>
            validationSchema.isValidSync(values.initialValues)
          }
          initialValues={initialValues}
          validationSchema={validationSchema}
          onSubmit={data => {}}
        >
          {({ errors, values, setFieldValue, isValid, ...rest }) => {
            const getEngine = (): ExecutionEngine => values.engine
            return (
              <Form>
                {requestStatus === RequestStatus.ERROR && error && (
                  <ErrorCallout error={error} />
                )}
                {/* Owner ID Field */}
                <FormGroup
                  label={helpers.ownerIdFieldSpec.label}
                  helperText={helpers.ownerIdFieldSpec.description}
                >
                  <FastField
                    name={helpers.ownerIdFieldSpec.name}
                    value={values.owner_id}
                    className={Classes.INPUT}
                  />
                  {errors.owner_id && (
                    <FieldError>{errors.owner_id}</FieldError>
                  )}
                </FormGroup>
                <div className="flotilla-form-section-divider" />
                {/* Engine Type Field */}
                <RadioGroup
                  inline
                  label="Engine Type"
                  onChange={(evt: React.FormEvent<HTMLInputElement>) => {
                    setFieldValue("engine", evt.currentTarget.value)
                  }}
                  selectedValue={values.engine}
                >
                  <Radio label="ECS" value={ExecutionEngine.LOCAL} />
                </RadioGroup>
                <div className="flotilla-form-section-divider" />

                {/*
                Cluster Field. Note: this is a "Field" rather than a
                "FastField" as it needs to re-render when value.engine is
                updated.
              */}

                {/* CPU Field */}
                <FormGroup
                  label={helpers.cpuFieldSpec.label}
                  helperText={helpers.cpuFieldSpec.description}
                >
                  <FastField
                    type="number"
                    name={helpers.cpuFieldSpec.name}
                    className={Classes.INPUT}
                    min="512"
                  />
                  {errors.cpu && <FieldError>{errors.cpu}</FieldError>}
                </FormGroup>

                {/* Memory Field */}
                <FormGroup
                  label={helpers.memoryFieldSpec.label}
                  helperText={helpers.memoryFieldSpec.description}
                >
                  <FastField
                    type="number"
                    name={helpers.memoryFieldSpec.name}
                    className={Classes.INPUT}
                  />
                  {errors.memory && <FieldError>{errors.memory}</FieldError>}
                </FormGroup>
                <div className="flotilla-form-section-divider" />
                <EnvFieldArray />
              </Form>
            )
          }}
        </Formik>
        <div className="flotilla-form-section-divider" />
        <JSONSchemaForm
          schema={template.schema}
          onSubmit={({ formData }) => {
            this.onSubmit(formData)
          }}
          onError={() => console.log("errors")}
          FieldTemplate={FieldTemplate}
          ArrayFieldTemplate={ArrayFieldTemplate}
          widgets={{
            BaseInput: props => {
              return (
                <input
                  className="bp3-input"
                  value={props.value}
                  required={props.required}
                  onChange={evt => {
                    props.onChange(evt.target.value)
                  }}
                />
              )
            },
          }}
        >
          <Button
            intent={Intent.PRIMARY}
            type="submit"
            disabled={isLoading}
            style={{ marginTop: 24 }}
            large
            fill
          >
            Submit
          </Button>
        </JSONSchemaForm>
      </div>
    )
  }
}

const Connected: React.FunctionComponent<RouteComponentProps> = ({
  history,
}) => {
  return (
    <Request<Run, { templateID: string; data: TemplateExecutionRequest }>
      requestFn={api.runTemplate}
      shouldRequestOnMount={false}
      onSuccess={(data: Run) => {
        Toaster.show({
          message: `Run ${data.run_id} submitted successfully!`,
          intent: Intent.SUCCESS,
        })
        history.push(`/runs/${data.run_id}`)
      }}
      onFailure={() => {
        Toaster.show({
          message: "An error occurred.",
          intent: Intent.DANGER,
        })
      }}
    >
      {requestProps => (
        <TemplateContext.Consumer>
          {(ctx: TemplateCtx) => {
            switch (ctx.requestStatus) {
              case RequestStatus.ERROR:
                return <ErrorCallout error={ctx.error} />
              case RequestStatus.READY:
                if (ctx.data) {
                  const initialValues: TemplateExecutionRequest = getInitialValuesForTemplateRun()
                  return (
                    <RunForm
                      templateID={ctx.templateID}
                      initialValues={initialValues}
                      template={ctx.data}
                      {...requestProps}
                    />
                  )
                }
                break
              case RequestStatus.NOT_READY:
              default:
                return <Spinner />
            }
          }}
        </TemplateContext.Consumer>
      )}
    </Request>
  )
}

export default Connected

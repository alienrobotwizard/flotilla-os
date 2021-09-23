import axios, { AxiosInstance, AxiosError, AxiosResponse } from "axios"
import * as qs from "qs"
import { has, omit, Omit } from "lodash"
import {
  HTTPMethod,
  CreateTaskPayload,
  RequestArgs,
  Run,
  ListRunParams,
  ListRunResponse,
  RunLog,
  LaunchRequestV2,
  Task,
  ListTaskResponse,
  ListTaskRunsResponse,
  UpdateTaskPayload,
  ListTaskParams,
  ListTaskRunsParams,
  ListClustersResponse,
  ListGroupsResponse,
  ListTagsResponse,
  ListRunEventsResponse,
  RunLogRaw,
  ListTemplateParams,
  ListTemplateResponse,
  Template,
  TemplateExecutionRequest,
  ListTemplateHistoryParams,
  ListTemplateHistoryResponse,
} from "../types"

interface IInitOpts {
  baseURL: string
  headers?: object
}

class FlotillaClient {
  private axios: AxiosInstance

  constructor({ baseURL, headers = {} }: IInitOpts) {
    this.axios = axios.create({
      baseURL,
      headers,
      // Note: this is the array format that the Flotilla server accepts.
      paramsSerializer: params =>
        qs.stringify(params, { arrayFormat: "repeat" }),
    })
  }

  /** Requests list of runs. */
  public listRun = ({
    params,
  }: {
    params: ListRunParams
  }): Promise<ListRunResponse> =>
    this.request<ListRunResponse>({
      method: HTTPMethod.GET,
      url: `/history`,
      params,
    })

  /** Requests a single run. */
  public getRun = ({ runID }: { runID: string }): Promise<Run> =>
    this.request<Run>({
      method: HTTPMethod.GET,
      url: `/history/${runID}`,
    })

  /** Requests the logs of a single run. */
  public getRunLog = ({
    runID,
    lastSeen = "",
  }: {
    runID: string
    lastSeen?: string
  }): Promise<RunLog> =>
    this.request<RunLog>({
      method: HTTPMethod.GET,
      url: `/history/${runID}/logs`,
      params: { last_seen: lastSeen },
    })

  /** Requests the logs of a single run. */
  public getRunLogRaw = ({ runID }: { runID: string }): Promise<RunLogRaw> =>
    this.request<RunLogRaw>({
      method: HTTPMethod.GET,
      url: `/history/${runID}/logs`,
      params: { raw_text: true },
    })


  /** Requests available clusters. */
  public listClusters = (): Promise<ListClustersResponse> =>
    this.request<ListClustersResponse>({
      method: HTTPMethod.GET,
      url: `/v6/clusters`,
    })

  /** Requests available groups. */
  public listGroups = (): Promise<ListGroupsResponse> =>
    this.request<ListGroupsResponse>({
      method: HTTPMethod.GET,
      url: `/v6/groups`,
      params: { offset: 0, limit: 10000 },
    })

  /** Requests available tags. */
  public listTags = (): Promise<ListTagsResponse> =>
    this.request<ListTagsResponse>({
      method: HTTPMethod.GET,
      url: `/v6/tags`,
      params: { offset: 0, limit: 10000 },
    })

  /** Requests available tags. */
  public listRunEvents = (runID: string): Promise<ListRunEventsResponse> =>
    this.request<ListRunEventsResponse>({
      method: HTTPMethod.GET,
      url: `/v6/${runID}/events`,
    })

  /** Requests a list of task definitions. */
  public listTemplates = ({
    params,
  }: {
    params: ListTemplateParams
  }): Promise<ListTemplateResponse> =>
    this.request<ListTemplateResponse>({
      method: HTTPMethod.GET,
      url: `/template`,
      params,
    })

  /** Requests a task definition. */
  public getTemplate = ({
    templateID,
  }: {
    templateID: string
  }): Promise<Template> =>
    this.request<Template>({
      method: HTTPMethod.GET,
      url: `/template/${templateID}`,
    })

  /** Runs a task. */
  public runTemplate = ({
    templateID,
    data,
  }: {
    templateID: string
    data: TemplateExecutionRequest
  }): Promise<Run> => {
    return this.request<Run>({
      method: HTTPMethod.PUT,
      url: `/template/${templateID}/execute`,
      data,
    })
  }

  /** Requests a task definition's history. */
  public listTemplateHistoryByTemplateID = ({
    templateID,
    params,
  }: {
    templateID: string
    params: ListTemplateHistoryParams
  }): Promise<ListTemplateHistoryResponse> =>
    this.request<ListTemplateHistoryResponse>({
      method: HTTPMethod.GET,
      url: `/template/${templateID}/history`,
      params,
    })

  /** Returns a new Promise that sends an HTTP request when invoked. */
  private request<T>({ method, url, params, data }: RequestArgs): Promise<T> {
    return new Promise((resolve, reject) => {
      this.axios
        .request({ url, method, params, data })
        .then((res: AxiosResponse) => {
          resolve(res.data as T)
        })
        .catch((error: AxiosError) => {
          reject(error)
        })
    })
  }
}

export default FlotillaClient

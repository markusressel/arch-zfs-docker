export interface KubectlCommand {
  label: string
  command: string
}

export function kubectlCommands(namespace: string, jobName: string): KubectlCommand[] {
  return [
    { label: 'Follow logs', command: `kubectl -n ${namespace} logs -f job/${jobName}` },
    { label: 'Job status', command: `kubectl -n ${namespace} describe job ${jobName}` },
    { label: 'Pods of this job', command: `kubectl -n ${namespace} get pods -l job-name=${jobName}` },
    { label: 'Delete job', command: `kubectl -n ${namespace} delete job ${jobName}` },
  ]
}

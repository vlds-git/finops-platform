{{/*
Generic deployment template for backend and ML services.
Usage: include "finops.deployment" (dict "Root" . "Name" "ingestion" "Component" "ingestion" "ValuesKey" "ingestion" "Port" 8081 "Env" (list ...))
*/}}
{{- define "finops.deployment" -}}
{{- $root := .Root -}}
{{- $name := .Name -}}
{{- $component := .Component -}}
{{- $values := index $root.Values .ValuesKey -}}
{{- $port := .Port -}}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "finops.fullname" $root }}-{{ $name }}
  labels:
    {{- include "finops.labels" $root | nindent 4 }}
    app.kubernetes.io/component: {{ $component }}
spec:
  replicas: {{ $values.replicaCount | default $root.Values.replicaCount }}
  selector:
    matchLabels:
      {{- include "finops.selectorLabels" $root | nindent 6 }}
      app.kubernetes.io/component: {{ $component }}
  template:
    metadata:
      labels:
        {{- include "finops.selectorLabels" $root | nindent 8 }}
        app.kubernetes.io/component: {{ $component }}
    spec:
      containers:
        - name: {{ $name }}
          image: {{ include "finops.image" (dict "Values" $root.Values "repository" $values.image.repository "tag" $values.image.tag) | quote }}
          imagePullPolicy: {{ $root.Values.image.pullPolicy }}
          ports:
            - name: http
              containerPort: {{ $port }}
              protocol: TCP
          env:
            - name: PORT
              value: "{{ $port }}"
            {{- range .Env }}
            - name: {{ .name }}
              value: {{ .value | quote }}
            {{- end }}
          livenessProbe:
            httpGet:
              path: /live
              port: http
            initialDelaySeconds: 10
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /ready
              port: http
            initialDelaySeconds: 5
            periodSeconds: 5
          resources:
            {{- toYaml $values.resources | nindent 12 }}
---
apiVersion: v1
kind: Service
metadata:
  name: {{ include "finops.fullname" $root }}-{{ $name }}
  labels:
    {{- include "finops.labels" $root | nindent 4 }}
    app.kubernetes.io/component: {{ $component }}
spec:
  type: {{ $values.service.type }}
  ports:
    - port: {{ $values.service.port }}
      targetPort: http
      protocol: TCP
      name: http
  selector:
    {{- include "finops.selectorLabels" $root | nindent 4 }}
    app.kubernetes.io/component: {{ $component }}
{{- end }}

window.addEventListener("DOMContentLoaded", function () {
  window.SwaggerUIBundle({
    url: "/openapi.yaml",
    dom_id: "#swagger-ui",
    deepLinking: true,
    displayRequestDuration: true,
    docExpansion: "list",
    filter: true,
    presets: [window.SwaggerUIBundle.presets.apis],
    supportedSubmitMethods: []
  });
});

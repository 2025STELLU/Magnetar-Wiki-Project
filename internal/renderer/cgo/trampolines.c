//go:build cgo && !nocgo

#include "ftml.h"
#include "_cgo_export.h"

void magnetar_fill_callbacks(FtmlCallbacks *out) {
  out->module_has_body = magnetarModuleHasBody;
  out->module_is_inline = magnetarModuleIsInline;
  out->render_module = magnetarRenderModule;
  out->render_user = magnetarRenderUser;
  out->get_i18n_message = magnetarGetI18nMessage;
  out->get_html_injected_code = magnetarGetHTMLInjectedCode;
  out->get_page_info = magnetarGetPageInfo;
  out->evaluate_expression = magnetarEvaluateExpression;
  out->normalize_page_name = magnetarNormalizePageName;
  out->include_pages = magnetarIncludePages;
  out->no_such_include = magnetarNoSuchInclude;
  out->next_include_level = magnetarNextIncludeLevel;
}

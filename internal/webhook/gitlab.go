package webhook

// Парсинг и обработка GitLab-событий (merge_request, push):
// верификация X-Gitlab-Token, извлечение issue identifier, upsert attachment,
// автостатус, linkback. Заполняется при реализации.

function testApiConnection() {
  try {
    const response = UrlFetchApp.fetch(`${API_BASE_URL}/ping`);
    if (response.getResponseCode() === 200 && response.getContentText() === 'pong') {
      Logger.log('✅ API доступен');
      return true;
    }
  } catch (error) {
    Logger.log('❌ API недоступен: ' + error.toString());
  }
  return false;
}

function getUserInfo() {
  const token = ensureAuthenticated();
  Logger.log('Текущий JWT токен активен');
  return { authenticated: true, token: token.substring(0, 20) + '...' };
}

function syncData() {
  try {
    Logger.log('Начало синхронизации данных...');

    exportBudgetsToSheet('Бюджеты');

    const importResult = importTransactionsFromSheet('Транзакции');

    exportTransactionsToSheet('Все транзакции');

    const now = new Date();
    const firstDay = new Date(now.getFullYear(), now.getMonth(), 1);
    const lastDay = new Date(now.getFullYear(), now.getMonth() + 1, 0);

    const fromDate = Utilities.formatDate(firstDay, Session.getScriptTimeZone(), 'yyyy-MM-dd');
    const toDate = Utilities.formatDate(lastDay, Session.getScriptTimeZone(), 'yyyy-MM-dd');

    exportReportToSheet(fromDate, toDate, 'Отчёт текущий месяц');

    Logger.log('✅ Синхронизация завершена успешно');
    return {
      imported: importResult,
      exported: true,
      report: { from: fromDate, to: toDate }
    };

  } catch (error) {
    Logger.log('❌ Ошибка синхронизации: ' + error.toString());
    throw error;
  }
}


function formatBudgetsSheet(sheetName = 'Бюджеты') {
  const sheet = SpreadsheetApp.getActiveSpreadsheet().getSheetByName(sheetName);
  if (!sheet) return;

  const range = sheet.getDataRange();

  range.offset(0, 0, 1, 4).setFontWeight('bold');

  sheet.getRange('A:A').setBackground('#E8F5E8');
  sheet.getRange('B:B').setBackground('#FFF3E0');
  sheet.getRange('C:C').setBackground('#E3F2FD');
  sheet.getRange('D:D').setBackground('#F3E5F5');

  sheet.autoResizeColumns(1, 4);

  sheet.setFrozenRows(1);

  Logger.log('Форматирование листа "' + sheetName + '" завершено');
}

function formatTransactionsSheet(sheetName = 'Все транзакции') {
  const sheet = SpreadsheetApp.getActiveSpreadsheet().getSheetByName(sheetName);
  if (!sheet) return;

  const range = sheet.getDataRange();

  range.offset(0, 0, 1, 5).setFontWeight('bold');

  const dateColumn = sheet.getRange('B:B');
  dateColumn.setNumberFormat('yyyy-mm-dd');

  const amountColumn = sheet.getRange('E:E');
  amountColumn.setNumberFormat('#,##0.00 ₽');

  sheet.getRange('A:A').setBackground('#F5F5F5');
  sheet.getRange('B:B').setBackground('#E8F5E8');
  sheet.getRange('C:C').setBackground('#E3F2FD');
  sheet.getRange('D:D').setBackground('#FFF3E0');
  sheet.getRange('E:E').setBackground('#F3E5F5');

  sheet.autoResizeColumns(1, 5);

  sheet.setFrozenRows(1);

  Logger.log('Форматирование листа "' + sheetName + '" завершено');
}

function formatReportSheet(sheetName = 'Отчёт текущий месяц') {
  const sheet = SpreadsheetApp.getActiveSpreadsheet().getSheetByName(sheetName);
  if (!sheet) return;

  const data = sheet.getDataRange().getValues();
  if (data.length < 2) return;

  const summaryRange = sheet.getRange(2, 2, data.length - 4, 1);
  summaryRange.setNumberFormat('#,##0.00 ₽');

  const totalsStartRow = data.length - 2;
  const totalsRange = sheet.getRange(totalsStartRow, 2, 3, 1);
  totalsRange.setNumberFormat('#,##0.00 ₽');
  totalsRange.setFontWeight('bold');

  sheet.getRange(totalsStartRow, 1, 3, 2).setBackground('#FFF9C4');

  sheet.autoResizeColumns(1, 2);

  Logger.log('Форматирование листа "' + sheetName + '" завершено');
}


function sendNotificationEmail(subject, body) {
  const userEmail = Session.getActiveUser().getEmail();
  GmailApp.sendEmail(userEmail, subject, body);
}

function checkBudgetsAndNotify() {
  try {
    const budgets = getBudgets();
    const notifications = [];

    for (const budget of budgets) {
      const usagePercent = ((budget.limit - budget.remaining) / budget.limit) * 100;

      if (usagePercent >= 90) {
        notifications.push({
          category: budget.category,
          usage: usagePercent.toFixed(1),
          remaining: budget.remaining
        });
      }
    }

    if (notifications.length > 0) {
      let message = '⚠️ Предупреждение о превышении бюджета:\n\n';

      for (const notif of notifications) {
        message += `• ${notif.category}: использовано ${notif.usage}%, осталось ${notif.remaining} ₽\n`;
      }

      sendNotificationEmail('Предупреждение о бюджете', message);
      Logger.log('Отправлено уведомление о бюджете');
    }

  } catch (error) {
    Logger.log('Ошибка проверки бюджетов: ' + error.toString());
  }
}

function setupDailyBudgetCheck() {
  const triggers = ScriptApp.getProjectTriggers();
  triggers.forEach(trigger => {
    if (trigger.getHandlerFunction() === 'checkBudgetsAndNotify') {
      ScriptApp.deleteTrigger(trigger);
    }
  });

  ScriptApp.newTrigger('checkBudgetsAndNotify')
    .timeBased()
    .everyDays(1)
    .atHour(9)
    .create();

  Logger.log('Настроена ежедневная проверка бюджетов');
}

function setupWeeklySync() {
  const triggers = ScriptApp.getProjectTriggers();
  triggers.forEach(trigger => {
    if (trigger.getHandlerFunction() === 'syncData') {
      ScriptApp.deleteTrigger(trigger);
    }
  });

  ScriptApp.newTrigger('syncData')
    .timeBased()
    .onWeekDay(ScriptApp.WeekDay.MONDAY)
    .atHour(8)
    .create();

  Logger.log('Настроена еженедельная синхронизация данных');
}

function clearAllTriggers() {
  const triggers = ScriptApp.getProjectTriggers();
  triggers.forEach(trigger => ScriptApp.deleteTrigger(trigger));
  Logger.log('Все триггеры удалены');
}

function createDashboard() {
  const spreadsheet = SpreadsheetApp.getActiveSpreadsheet();

  let dashboard = spreadsheet.getSheetByName('Дашборд');
  if (!dashboard) {
    dashboard = spreadsheet.insertSheet('Дашборд');
  }

  dashboard.clear();

  dashboard.getRange('A1').setValue('Финансовый Дашборд');
  dashboard.getRange('A1').setFontSize(16).setFontWeight('bold');

  const budgets = getBudgets();
  const report = getReportSummary(
    Utilities.formatDate(new Date(new Date().getFullYear(), 0, 1), Session.getScriptTimeZone(), 'yyyy-MM-dd'),
    Utilities.formatDate(new Date(), Session.getScriptTimeZone(), 'yyyy-MM-dd')
  );

  dashboard.getRange('A3').setValue('Дата обновления:');
  dashboard.getRange('B3').setValue(new Date());

  dashboard.getRange('A5').setValue('ОБЩИЕ ПОКАЗАТЕЛИ');
  dashboard.getRange('A5:B5').merge().setFontWeight('bold').setBackground('#E8F5E8');

  dashboard.getRange('A6').setValue('Общий бюджет:');
  dashboard.getRange('B6').setValue(report.total_budget || 0).setNumberFormat('#,##0.00 ₽');

  dashboard.getRange('A7').setValue('Общие расходы:');
  dashboard.getRange('B7').setValue(report.total_expenses || 0).setNumberFormat('#,##0.00 ₽');

  dashboard.getRange('A8').setValue('Процент использования:');
  dashboard.getRange('B8').setValue((report.budget_usage_percent || 0) + '%');

  dashboard.getRange('A10').setValue('СТАТУС БЮДЖЕТОВ');
  dashboard.getRange('A10:D10').merge().setFontWeight('bold').setBackground('#E3F2FD');

  dashboard.getRange('A11:D11').setValues([['Категория', 'Лимит', 'Потрачено', 'Статус']]);
  dashboard.getRange('A11:D11').setFontWeight('bold');

  if (budgets && budgets.length > 0) {
    for (let i = 0; i < budgets.length; i++) {
      const budget = budgets[i];
      const spent = budget.limit - budget.remaining;
      const usagePercent = (spent / budget.limit) * 100;

      let status = '✅ Норма';
      let statusColor = '#E8F5E8';

      if (usagePercent >= 90) {
        status = '⚠️ Предупреждение';
        statusColor = '#FFF3E0';
      } else if (usagePercent >= 100) {
        status = '❌ Превышен';
        statusColor = '#FFEBEE';
      }

      const row = i + 12;
      dashboard.getRange(row, 1, 1, 4).setValues([[
        budget.category,
        budget.limit,
        spent,
        status
      ]]);

      dashboard.getRange(row, 4).setBackground(statusColor);
    }
  }

  dashboard.getRange('B:B').setNumberFormat('#,##0.00 ₽');
  dashboard.getRange('C:C').setNumberFormat('#,##0.00 ₽');
  dashboard.autoResizeColumns(1, 4);

  Logger.log('Дашборд создан');
}

function importFromCSV(fileName = 'transactions.csv') {
  try {
    const files = DriveApp.getFilesByName(fileName);
    if (!files.hasNext()) {
      throw new Error('Файл "' + fileName + '" не найден в Google Drive');
    }

    const file = files.next();
    const csvData = file.getBlob().getDataAsString();
    const lines = csvData.split('\n');

    if (lines.length < 2) {
      throw new Error('CSV файл пуст или содержит только заголовки');
    }

    const transactions = [];

    for (let i = 1; i < lines.length; i++) {
      const line = lines[i].trim();
      if (!line) continue;

      const columns = line.split(',');
      if (columns.length >= 4) {
        const amount = parseFloat(columns[1].trim());
        const category = columns[2].trim();
        const description = columns[3].trim();
        const date = columns[0].trim();

        if (!isNaN(amount) && category && date) {
          transactions.push({
            amount: amount,
            category: category,
            description: description,
            date: date
          });
        }
      }
    }

    if (transactions.length === 0) {
      throw new Error('Не найдено валидных транзакций в CSV файле');
    }

    Logger.log('Найдено транзакций в CSV: ' + transactions.length);
    return bulkImportTransactions(transactions);

  } catch (error) {
    Logger.log('Ошибка импорта из CSV: ' + error.toString());
    throw error;
  }
}

function createBackup() {
  try {
    const timestamp = Utilities.formatDate(new Date(), Session.getScriptTimeZone(), 'yyyy-MM-dd_HH-mm-ss');
    const backupFolderName = 'Finance Backup ' + timestamp;

    const backupFolder = DriveApp.createFolder(backupFolderName);

    const backupData = {
      timestamp: new Date().toISOString(),
      budgets: getBudgets(),
      transactions: getTransactions(),
      spreadsheet_url: SpreadsheetApp.getActiveSpreadsheet().getUrl()
    };

    const jsonFile = backupFolder.createFile(
      'backup_data.json',
      JSON.stringify(backupData, null, 2),
      MimeType.PLAIN_TEXT
    );

    const sheets = ['Транзакции', 'Бюджеты', 'Все транзакции'];
    sheets.forEach(sheetName => {
      const sheet = SpreadsheetApp.getActiveSpreadsheet().getSheetByName(sheetName);
      if (sheet) {
        const csvData = convertSheetToCSV(sheet);
        backupFolder.createFile(sheetName + '.csv', csvData, MimeType.CSV);
      }
    });

    Logger.log('Резервная копия создана: ' + backupFolder.getUrl());
    return backupFolder.getUrl();

  } catch (error) {
    Logger.log('Ошибка создания резервной копии: ' + error.toString());
    throw error;
  }
}

function convertSheetToCSV(sheet) {
  const data = sheet.getDataRange().getValues();
  let csv = '';

  for (let i = 0; i < data.length; i++) {
    const row = data[i];
    for (let j = 0; j < row.length; j++) {
      if (j > 0) csv += ',';
      csv += '"' + row[j].toString().replace(/"/g, '""') + '"';
    }
    csv += '\n';
  }

  return csv;
}

function onOpen() {
  const ui = SpreadsheetApp.getUi();
  ui.createMenu('Финансы API')
    .addItem('Авторизоваться', 'showLoginDialog')
    .addSeparator()
    .addItem('Импорт транзакций', 'importTransactionsFromSheet')
    .addItem('Экспорт бюджетов', 'exportBudgetsToSheet')
    .addItem('Экспорт транзакций', 'exportTransactionsToSheet')
    .addItem('Экспорт отчёта', 'showReportDialog')
    .addSeparator()
    .addSubMenu(ui.createMenu('Дополнительно')
      .addItem('Создать дашборд', 'createDashboard')
      .addItem('Синхронизировать данные', 'syncData')
      .addItem('Проверить API', 'testApiConnection')
      .addSeparator()
      .addItem('Создать резервную копию', 'createBackup')
      .addItem('Импорт из CSV', 'importFromCSV'))
    .addSubMenu(ui.createMenu('Автоматизация')
      .addItem('Настроить ежедневные уведомления', 'setupDailyBudgetCheck')
      .addItem('Настроить еженедельную синхронизацию', 'setupWeeklySync')
      .addSeparator()
      .addItem('Очистить все триггеры', 'clearAllTriggers'))
    .addSeparator()
    .addItem('Настройки', 'showSettingsDialog')
    .addToUi();
}


function completeSetupExample() {
  try {
    Logger.log('Начало полной настройки...');

    if (!testApiConnection()) {
      throw new Error('API недоступен. Проверьте настройки.');
    }

    Logger.log('Выполните авторизацию через меню: Финансы API → Авторизоваться');

    createTransactionSheetTemplate();
    createDashboard();

    setupDailyBudgetCheck();
    setupWeeklySync();

    formatBudgetsSheet();
    formatTransactionsSheet();

    Logger.log('✅ Полная настройка завершена!');
    Logger.log('Следующие шаги:');
    Logger.log('   1. Авторизуйтесь в системе');
    Logger.log('   2. Добавьте бюджеты');
    Logger.log('   3. Внесите транзакции в лист "Транзакции"');
    Logger.log('   4. Используйте меню для синхронизации');

  } catch (error) {
    Logger.log('❌ Ошибка настройки: ' + error.toString());
    Logger.log('Проверьте:');
    Logger.log('   - Доступность API');
    Logger.log('   - Правильность API URL');
    Logger.log('   - Регистрацию пользователя');
  }
}

function createTransactionSheetTemplate() {
  const spreadsheet = SpreadsheetApp.getActiveSpreadsheet();

  let sheet = spreadsheet.getSheetByName('Транзакции');
  if (!sheet) {
    sheet = spreadsheet.insertSheet('Транзакции');
  }

  sheet.clear();

  sheet.getRange('A1:D1').setValues([['Дата', 'Сумма', 'Категория', 'Описание']]);
  sheet.getRange('A1:D1').setFontWeight('bold').setBackground('#E8F5E8');

  const examples = [
    ['2025-01-15', 450, 'еда', 'Обед в кафе'],
    ['2025-01-16', 1200, 'транспорт', 'Билет на поезд'],
    ['2025-01-17', 2500, 'жилье', 'Аренда квартиры'],
    ['2025-01-18', 150, 'развлечения', 'Кино']
  ];

  sheet.getRange(2, 1, examples.length, 4).setValues(examples);

  sheet.getRange('A:A').setNumberFormat('yyyy-mm-dd');
  sheet.getRange('B:B').setNumberFormat('#,##0.00 ₽');
  sheet.autoResizeColumns(1, 4);
  sheet.setFrozenRows(1);

  sheet.getRange('F1').setValue('Инструкция:');
  sheet.getRange('F2').setValue('1. Заполняйте таблицу новыми транзакциями');
  sheet.getRange('F3').setValue('2. Используйте меню "Финансы API → Импорт транзакций"');
  sheet.getRange('F4').setValue('3. Для экспорта данных используйте соответствующие пункты меню');

  Logger.log('Создан шаблон листа транзакций');
}
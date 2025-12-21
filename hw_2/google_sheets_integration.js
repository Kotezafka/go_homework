const API_BASE_URL = 'https://your-api-domain.com';

let JWT_TOKEN = null;

function registerUser(email, password, name) {
  const url = `${API_BASE_URL}/auth/register`;
  const payload = {
    email: email,
    password: password,
    name: name
  };

  const options = {
    method: 'post',
    contentType: 'application/json',
    payload: JSON.stringify(payload),
    muteHttpExceptions: true
  };

  try {
    const response = UrlFetchApp.fetch(url, options);
    const result = JSON.parse(response.getContentText());

    if (response.getResponseCode() === 200) {
      Logger.log('Пользователь зарегистрирован: ' + result.user_id);
      return result;
    } else {
      throw new Error('Ошибка регистрации: ' + response.getContentText());
    }
  } catch (error) {
    Logger.log('Ошибка: ' + error.toString());
    throw error;
  }
}

function loginUser(email, password) {
  const url = `${API_BASE_URL}/auth/login`;
  const payload = {
    email: email,
    password: password
  };

  const options = {
    method: 'post',
    contentType: 'application/json',
    payload: JSON.stringify(payload),
    muteHttpExceptions: true
  };

  try {
    const response = UrlFetchApp.fetch(url, options);
    const result = JSON.parse(response.getContentText());

    if (response.getResponseCode() === 200) {
      JWT_TOKEN = result.token;
      PropertiesService.getScriptProperties().setProperty('JWT_TOKEN', JWT_TOKEN);
      Logger.log('Авторизация успешна, токен сохранен');
      return result;
    } else {
      throw new Error('Ошибка авторизации: ' + response.getContentText());
    }
  } catch (error) {
    Logger.log('Ошибка: ' + error.toString());
    throw error;
  }
}

function loadToken() {
  if (!JWT_TOKEN) {
    JWT_TOKEN = PropertiesService.getScriptProperties().getProperty('JWT_TOKEN');
  }
  return JWT_TOKEN;
}

function ensureAuthenticated() {
  const token = loadToken();
  if (!token) {
    throw new Error('Требуется авторизация. Выполните loginUser() сначала.');
  }
  return token;
}

function createBudget(category, limit) {
  const token = ensureAuthenticated();
  const url = `${API_BASE_URL}/api/budgets`;

  const payload = {
    category: category,
    limit: limit
  };

  const options = {
    method: 'post',
    contentType: 'application/json',
    headers: {
      'Authorization': `Bearer ${token}`
    },
    payload: JSON.stringify(payload),
    muteHttpExceptions: true
  };

  try {
    const response = UrlFetchApp.fetch(url, options);
    const result = JSON.parse(response.getContentText());

    if (response.getResponseCode() === 201) {
      Logger.log('Бюджет создан: ' + category);
      return result;
    } else {
      throw new Error('Ошибка создания бюджета: ' + response.getContentText());
    }
  } catch (error) {
    Logger.log('Ошибка: ' + error.toString());
    throw error;
  }
}

function getBudgets() {
  const token = ensureAuthenticated();
  const url = `${API_BASE_URL}/api/budgets`;

  const options = {
    method: 'get',
    headers: {
      'Authorization': `Bearer ${token}`
    },
    muteHttpExceptions: true
  };

  try {
    const response = UrlFetchApp.fetch(url, options);
    const result = JSON.parse(response.getContentText());

    if (response.getResponseCode() === 200) {
      Logger.log('Получено бюджетов: ' + result.length);
      return result;
    } else {
      throw new Error('Ошибка получения бюджетов: ' + response.getContentText());
    }
  } catch (error) {
    Logger.log('Ошибка: ' + error.toString());
    throw error;
  }
}

function createTransaction(amount, category, description, date) {
  const token = ensureAuthenticated();
  const url = `${API_BASE_URL}/api/transactions`;

  const payload = {
    amount: amount,
    category: category,
    description: description,
    date: date
  };

  const options = {
    method: 'post',
    contentType: 'application/json',
    headers: {
      'Authorization': `Bearer ${token}`
    },
    payload: JSON.stringify(payload),
    muteHttpExceptions: true
  };

  try {
    const response = UrlFetchApp.fetch(url, options);
    const result = JSON.parse(response.getContentText());

    if (response.getResponseCode() === 201) {
      Logger.log('Транзакция создана: ' + description);
      return result;
    } else {
      throw new Error('Ошибка создания транзакции: ' + response.getContentText());
    }
  } catch (error) {
    Logger.log('Ошибка: ' + error.toString());
    throw error;
  }
}

function getTransactions() {
  const token = ensureAuthenticated();
  const url = `${API_BASE_URL}/api/transactions`;

  const options = {
    method: 'get',
    headers: {
      'Authorization': `Bearer ${token}`
    },
    muteHttpExceptions: true
  };

  try {
    const response = UrlFetchApp.fetch(url, options);
    const result = JSON.parse(response.getContentText());

    if (response.getResponseCode() === 200) {
      Logger.log('Получено транзакций: ' + result.length);
      return result;
    } else {
      throw new Error('Ошибка получения транзакций: ' + response.getContentText());
    }
  } catch (error) {
    Logger.log('Ошибка: ' + error.toString());
    throw error;
  }
}

function bulkImportTransactions(transactions) {
  const token = ensureAuthenticated();
  const url = `${API_BASE_URL}/api/transactions/bulk`;

  const options = {
    method: 'post',
    contentType: 'application/json',
    headers: {
      'Authorization': `Bearer ${token}`
    },
    payload: JSON.stringify(transactions),
    muteHttpExceptions: true
  };

  try {
    const response = UrlFetchApp.fetch(url, options);
    const result = JSON.parse(response.getContentText());

    if (response.getResponseCode() === 200) {
      Logger.log('Импорт завершен: принято ' + result.accepted + ', отклонено ' + result.rejected);
      return result;
    } else {
      throw new Error('Ошибка импорта: ' + response.getContentText());
    }
  } catch (error) {
    Logger.log('Ошибка: ' + error.toString());
    throw error;
  }
}

function getReportSummary(fromDate, toDate) {
  const token = ensureAuthenticated();
  const url = `${API_BASE_URL}/api/reports/summary?from=${fromDate}&to=${toDate}`;

  const options = {
    method: 'get',
    headers: {
      'Authorization': `Bearer ${token}`
    },
    muteHttpExceptions: true
  };

  try {
    const response = UrlFetchApp.fetch(url, options);
    const result = JSON.parse(response.getContentText());

    if (response.getResponseCode() === 200) {
      Logger.log('Отчёт получен');
      return result;
    } else {
      throw new Error('Ошибка получения отчёта: ' + response.getContentText());
    }
  } catch (error) {
    Logger.log('Ошибка: ' + error.toString());
    throw error;
  }
}

function importTransactionsFromSheet(sheetName = 'Транзакции') {
  const sheet = SpreadsheetApp.getActiveSpreadsheet().getSheetByName(sheetName);
  if (!sheet) {
    throw new Error('Лист "' + sheetName + '" не найден');
  }

  const data = sheet.getDataRange().getValues();

  if (data.length <= 1) {
    Logger.log('Нет данных для импорта');
    return;
  }

  const transactions = [];

  for (let i = 1; i < data.length; i++) {
    const row = data[i];
    if (row.length >= 4 && row[0] && row[1] && row[2]) {
      const date = Utilities.formatDate(row[0], Session.getScriptTimeZone(), 'yyyy-MM-dd');
      const amount = parseFloat(row[1]);
      const category = row[2].toString();
      const description = row[3] ? row[3].toString() : '';

      if (!isNaN(amount) && category) {
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
    Logger.log('Нет валидных транзакций для импорта');
    return;
  }

  Logger.log('Найдено транзакций для импорта: ' + transactions.length);
  return bulkImportTransactions(transactions);
}

function exportBudgetsToSheet(sheetName = 'Бюджеты') {
  const budgets = getBudgets();

  let sheet = SpreadsheetApp.getActiveSpreadsheet().getSheetByName(sheetName);
  if (!sheet) {
    sheet = SpreadsheetApp.getActiveSpreadsheet().insertSheet(sheetName);
  }

  sheet.clear();

  sheet.getRange(1, 1, 1, 4).setValues([['Категория', 'Лимит', 'Остаток', 'Период']]);

  if (budgets && budgets.length > 0) {
    const data = budgets.map(budget => [
      budget.category,
      budget.limit,
      budget.remaining,
      budget.period
    ]);
    sheet.getRange(2, 1, data.length, 4).setValues(data);
  }

  Logger.log('Бюджеты экспортированы в лист: ' + sheetName);
}

function exportReportToSheet(fromDate, toDate, sheetName = 'Отчёт') {
  const report = getReportSummary(fromDate, toDate);

  let sheet = SpreadsheetApp.getActiveSpreadsheet().getSheetByName(sheetName);
  if (!sheet) {
    sheet = SpreadsheetApp.getActiveSpreadsheet().insertSheet(sheetName);
  }

  sheet.clear();

  sheet.getRange(1, 1, 1, 2).setValues([['Категория', 'Сумма']]);

  if (report && report.summaries && report.summaries.length > 0) {
    const data = report.summaries.map(item => [
      item.category,
      item.total
    ]);
    sheet.getRange(2, 1, data.length, 2).setValues(data);

    const lastRow = data.length + 3;
    sheet.getRange(lastRow, 1, 3, 2).setValues([
      ['Общий бюджет', report.total_budget || 0],
      ['Общие расходы', report.total_expenses || 0],
      ['Процент использования', (report.budget_usage_percent || 0) + '%']
    ]);
  }

  Logger.log('Отчёт экспортирован в лист: ' + sheetName);
}

function exportTransactionsToSheet(sheetName = 'Экспорт транзакций') {
  const transactions = getTransactions();

  let sheet = SpreadsheetApp.getActiveSpreadsheet().getSheetByName(sheetName);
  if (!sheet) {
    sheet = SpreadsheetApp.getActiveSpreadsheet().insertSheet(sheetName);
  }

  sheet.clear();

  sheet.getRange(1, 1, 1, 5).setValues([['ID', 'Дата', 'Категория', 'Описание', 'Сумма']]);

  if (transactions && transactions.length > 0) {
    const data = transactions.map(tx => [
      tx.id,
      tx.date,
      tx.category,
      tx.description,
      tx.amount
    ]);
    sheet.getRange(2, 1, data.length, 5).setValues(data);
  }

  Logger.log('Транзакции экспортированы в лист: ' + sheetName);
}

function generateTransactionsCSV(transactions) {
  if (!transactions || transactions.length === 0) {
    return 'amount,category,description,date\n';
  }

  let csv = 'amount,category,description,date\n';
  
  transactions.forEach(tx => {
    const amount = (tx.amount || 0).toFixed(2);
    const category = escapeCSVField(tx.category || '');
    const description = escapeCSVField(tx.description || '');
    const date = tx.date || '';
    
    csv += `${amount},${category},${description},${date}\n`;
  });

  return csv;
}

function generateReportCSV(report) {
  let csv = 'category,total\n';
  
  if (report && report.summaries && report.summaries.length > 0) {
    report.summaries.forEach(item => {
      const category = escapeCSVField(item.category || '');
      const total = (item.total || 0).toFixed(2);
      csv += `${category},${total}\n`;
    });
    
    csv += '\n';
    csv += `Total Budget,${(report.total_budget || 0).toFixed(2)}\n`;
    csv += `Total Expenses,${(report.total_expenses || 0).toFixed(2)}\n`;
    csv += `Budget Usage Percent,${(report.budget_usage_percent || 0).toFixed(2)}%\n`;
  }

  return csv;
}

function escapeCSVField(field) {
  if (field === null || field === undefined) {
    return '""';
  }
  
  const str = String(field);
  if (str.includes(',') || str.includes('"') || str.includes('\n')) {
    return '"' + str.replace(/"/g, '""') + '"';
  }
  return str;
}

function exportTransactionsToCSV(fileName = null) {
  const token = ensureAuthenticated();
  const transactions = getTransactions();
  
  if (!transactions || transactions.length === 0) {
    Logger.log('Нет транзакций для экспорта');
    return null;
  }

  const csvData = generateTransactionsCSV(transactions);
  
  if (!fileName) {
    const timestamp = Utilities.formatDate(new Date(), Session.getScriptTimeZone(), 'yyyy-MM-dd_HH-mm-ss');
    fileName = `transactions_${timestamp}.csv`;
  }

  try {
    const file = DriveApp.createFile(fileName, csvData, MimeType.CSV);
    Logger.log('CSV файл создан: ' + file.getName());
    Logger.log('URL файла: ' + file.getUrl());
    return file.getUrl();
  } catch (error) {
    Logger.log('Ошибка создания CSV файла: ' + error.toString());
    throw error;
  }
}

function exportReportToCSV(fromDate, toDate, fileName = null) {
  const token = ensureAuthenticated();
  const report = getReportSummary(fromDate, toDate);
  
  if (!report) {
    Logger.log('Не удалось получить отчёт');
    return null;
  }

  const csvData = generateReportCSV(report);
  
  if (!fileName) {
    const timestamp = Utilities.formatDate(new Date(), Session.getScriptTimeZone(), 'yyyy-MM-dd_HH-mm-ss');
    fileName = `report_${fromDate}_${toDate}_${timestamp}.csv`;
  }

  try {
    const file = DriveApp.createFile(fileName, csvData, MimeType.CSV);
    Logger.log('CSV файл отчёта создан: ' + file.getName());
    Logger.log('URL файла: ' + file.getUrl());
    return file.getUrl();
  } catch (error) {
    Logger.log('Ошибка создания CSV файла отчёта: ' + error.toString());
    throw error;
  }
}

function exportTransactionsToCSVByDate(fromDate, toDate, fileName = null) {
  const token = ensureAuthenticated();
  const apiUrl = getApiUrl();
  
  const transactions = getTransactions();
  
  if (!transactions || transactions.length === 0) {
    Logger.log('Нет транзакций для экспорта');
    return null;
  }

  const filteredTransactions = transactions.filter(tx => {
    const txDate = tx.date;
    return txDate >= fromDate && txDate <= toDate;
  });

  if (filteredTransactions.length === 0) {
    Logger.log('Нет транзакций за указанный период');
    return null;
  }

  const csvData = generateTransactionsCSV(filteredTransactions);
  
  if (!fileName) {
    const timestamp = Utilities.formatDate(new Date(), Session.getScriptTimeZone(), 'yyyy-MM-dd_HH-mm-ss');
    fileName = `transactions_${fromDate}_${toDate}_${timestamp}.csv`;
  }

  try {
    const file = DriveApp.createFile(fileName, csvData, MimeType.CSV);
    Logger.log('CSV файл создан: ' + file.getName());
    Logger.log('URL файла: ' + file.getUrl());
    return file.getUrl();
  } catch (error) {
    Logger.log('Ошибка создания CSV файла: ' + error.toString());
    throw error;
  }
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
    .addSubMenu(ui.createMenu('CSV Экспорт')
      .addItem('Экспорт всех транзакций в CSV', 'exportTransactionsToCSV')
      .addItem('Экспорт отчёта в CSV', 'showReportCSVDialog'))
    .addSeparator()
    .addItem('Настройки', 'showSettingsDialog')
    .addToUi();
}

function showLoginDialog() {
  const html = HtmlService.createHtmlOutputFromFile('google_sheets_ui')
    .setWidth(550)
    .setHeight(450);

  SpreadsheetApp.getUi().showModalDialog(html, 'Авторизация');
}

function showReportDialog() {
  const html = HtmlService.createHtmlOutputFromFile('google_sheets_ui')
    .setWidth(550)
    .setHeight(450);

  PropertiesService.getScriptProperties().setProperty('dialog_context', 'report');

  SpreadsheetApp.getUi().showModalDialog(html, 'Экспорт отчёта');
}

function showReportCSVDialog() {
  const html = HtmlService.createHtmlOutput(`
    <div style="padding: 20px;">
      <h3>Экспорт отчёта в CSV</h3>
      <p>Введите период для экспорта:</p>
      <p>
        <label>От (YYYY-MM-DD):</label><br>
        <input type="text" id="fromDate" placeholder="2025-01-01" style="width: 100%; padding: 5px;">
      </p>
      <p>
        <label>До (YYYY-MM-DD):</label><br>
        <input type="text" id="toDate" placeholder="2025-12-31" style="width: 100%; padding: 5px;">
      </p>
      <p>
        <label>Имя файла (необязательно):</label><br>
        <input type="text" id="fileName" placeholder="report.csv" style="width: 100%; padding: 5px;">
      </p>
      <p>
        <button onclick="exportCSV()" style="padding: 10px 20px; background: #4285f4; color: white; border: none; cursor: pointer;">Экспортировать</button>
      </p>
      <div id="result" style="margin-top: 10px;"></div>
    </div>
    <script>
      function exportCSV() {
        const fromDate = document.getElementById('fromDate').value;
        const toDate = document.getElementById('toDate').value;
        const fileName = document.getElementById('fileName').value;
        const resultDiv = document.getElementById('result');
        
        if (!fromDate || !toDate) {
          resultDiv.innerHTML = '<p style="color: red;">Заполните даты!</p>';
          return;
        }
        
        resultDiv.innerHTML = '<p>Экспорт... Пожалуйста, подождите.</p>';
        
        google.script.run
          .withSuccessHandler(function(url) {
            resultDiv.innerHTML = '<p style="color: green;">✅ CSV файл создан!</p><p><a href="' + url + '" target="_blank">Открыть файл</a></p>';
          })
          .withFailureHandler(function(error) {
            resultDiv.innerHTML = '<p style="color: red;">Ошибка: ' + error.message + '</p>';
          })
          .exportReportToCSV(fromDate, toDate, fileName || null);
      }
    </script>
  `)
    .setWidth(400)
    .setHeight(400);

  SpreadsheetApp.getUi().showModalDialog(html, 'Экспорт отчёта в CSV');
}

function showSettingsDialog() {
  const html = HtmlService.createHtmlOutputFromFile('google_sheets_ui')
    .setWidth(550)
    .setHeight(450);

  PropertiesService.getScriptProperties().setProperty('dialog_context', 'settings');

  SpreadsheetApp.getUi().showModalDialog(html, 'Настройки');
}

function getDialogContext() {
  return PropertiesService.getScriptProperties().getProperty('dialog_context') || 'login';
}

function getApiUrl() {
  return PropertiesService.getScriptProperties().getProperty('API_BASE_URL') || API_BASE_URL;
}

function saveApiUrl(url) {
  PropertiesService.getScriptProperties().setProperty('API_BASE_URL', url);
  Logger.log('API URL сохранен: ' + url);
}


function exampleUsage() {
  try {
    createBudget('еда', 5000);
    createBudget('транспорт', 3000);
    createBudget('развлечения', 2000);

    createTransaction(450, 'еда', 'Обед в кафе', '2025-01-15');
    createTransaction(1200, 'транспорт', 'Билет на поезд', '2025-01-16');

    const report = getReportSummary('2025-01-01', '2025-12-31');
    Logger.log('Отчёт:', report);

  } catch (error) {
    Logger.log('Ошибка в примере: ' + error.toString());
  }
}
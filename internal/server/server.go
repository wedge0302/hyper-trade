package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"trade/internal/strategy"
)

// Server exposes a lightweight HTTP UI for monitoring and tuning the strategy.
type Server struct {
	addr     string
	settings *strategy.Settings
	metrics  *strategy.Metrics
	srv      *http.Server
}

// New constructs a monitoring server instance.
func New(addr string, settings *strategy.Settings, metrics *strategy.Metrics) *Server {
	s := &Server{
		addr:     addr,
		settings: settings,
		metrics:  metrics,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/params", s.handleUpdateParams)

	s.srv = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return s
}

// Run starts the HTTP server and blocks until the provided context is cancelled.
func (s *Server) Run(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.srv.Shutdown(shutdownCtx)
	}()

	if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	params := s.settings.Snapshot()
	snapshot := s.metrics.Snapshot()

	uptime := time.Duration(0)
	if !snapshot.Summary.StartTime.IsZero() {
		uptime = time.Since(snapshot.Summary.StartTime)
	}

	data := struct {
		Params   strategy.Params
		Snapshot strategy.DashboardSnapshot
		Updated  bool
		Uptime   time.Duration
	}{
		Params:   params,
		Snapshot: snapshot,
		Updated:  r.URL.Query().Get("updated") == "1",
		Uptime:   uptime,
	}

	if err := pageTemplate.Execute(w, data); err != nil {
		http.Error(w, fmt.Sprintf("render template: %v", err), http.StatusInternalServerError)
	}
}

func (s *Server) handleUpdateParams(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("parse form: %v", err), http.StatusBadRequest)
		return
	}

	amplitude, hasAmplitude, err := parseFloat(r.FormValue("amplitude"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	amount, hasAmount, err := parseFloat(r.FormValue("amount"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	takeProfit, hasTakeProfit, err := parseFloat(r.FormValue("take_profit"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stopLoss, hasStopLoss, err := parseFloat(r.FormValue("stop_loss"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	leverage, hasLeverage, err := parseFloat(r.FormValue("leverage"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ttl, hasTTL, err := parseDuration(r.FormValue("ttl"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	window, hasWindow, err := parseDuration(r.FormValue("window"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	initialBalance, hasInitialBalance, err := parseFloat(r.FormValue("initial_balance"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	updated, err := s.settings.Update(func(p *strategy.Params) error {
		if hasAmplitude {
			if amplitude <= 0 {
				return fmt.Errorf("amplitude must be positive")
			}
			p.AmplitudeThreshold = amplitude
		}

		if hasAmount {
			if amount <= 0 {
				return fmt.Errorf("amount must be positive")
			}
			p.OrderAmount = amount
		}

		if hasTakeProfit {
			if takeProfit <= 0 {
				return fmt.Errorf("take profit must be positive")
			}
			p.TakeProfitRatio = takeProfit
		}

		if hasStopLoss {
			if stopLoss <= 0 {
				return fmt.Errorf("stop loss must be positive")
			}
			p.StopLossRatio = stopLoss
		}

		if hasLeverage {
			if leverage <= 0 {
				return fmt.Errorf("leverage must be positive")
			}
			p.Leverage = leverage
		}

		if hasTTL {
			if ttl < 0 {
				return fmt.Errorf("ttl cannot be negative")
			}
			p.PositionTTL = ttl
		}

		if hasWindow {
			if window <= 0 {
				return fmt.Errorf("window must be positive")
			}
			p.AmplitudeWindow = window
		}

		if hasInitialBalance {
			if initialBalance < 0 {
				return fmt.Errorf("initial balance cannot be negative")
			}
			p.InitialBalance = initialBalance
		}

		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.metrics.SetInitialBalance(updated.InitialBalance)

	http.Redirect(w, r, "/?updated=1", http.StatusSeeOther)
}

func parseFloat(value string) (float64, bool, error) {
	if value == "" {
		return 0, false, nil
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false, fmt.Errorf("invalid float value: %w", err)
	}

	return parsed, true, nil
}

func parseDuration(value string) (time.Duration, bool, error) {
	if value == "" {
		return 0, false, nil
	}

	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, false, fmt.Errorf("invalid duration: %w", err)
	}

	return d, true, nil
}

var pageTemplate = template.Must(template.New("index").Funcs(template.FuncMap{
	"formatPercent": func(v float64) string {
		return fmt.Sprintf("%.3f%%", v*100)
	},
	"formatFloat": func(v float64) string {
		return fmt.Sprintf("%.4f", v)
	},
	"formatCurrency": func(v float64) string {
		return fmt.Sprintf("%.2f", v)
	},
	"formatTime": func(t time.Time) string {
		if t.IsZero() {
			return "-"
		}
		return t.Local().Format("2006-01-02 15:04:05")
	},
	"formatDuration": func(d time.Duration) string {
		if d <= 0 {
			return "-"
		}
		return d.Truncate(time.Second).String()
	},
	"calcReturn": func(profit, initial float64) float64 {
		if initial == 0 {
			return 0
		}
		return profit / initial
	},
	"toJSON": func(v any) template.JS {
		data, err := json.Marshal(v)
		if err != nil {
			return template.JS("[]")
		}
		return template.JS(data)
	},
}).Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <title>Hyper Trade Monitor</title>
  <style>
    :root {
      color-scheme: dark;
    }
    body {
      font-family: 'Inter', 'Segoe UI', Tahoma, sans-serif;
      margin: 0;
      padding: 2rem;
      background: radial-gradient(circle at top left, #1b2735, #090a0f);
      color: #e8f6ff;
    }
    h1 {
      color: #8ef6ff;
      margin-bottom: 1.5rem;
      text-shadow: 0 2px 10px rgba(142, 246, 255, 0.35);
    }
    table {
      width: 100%;
      border-collapse: collapse;
      margin-top: 1rem;
      color: #d4f1ff;
    }
    th, td {
      border: 1px solid rgba(142, 246, 255, 0.1);
      padding: 0.65rem 0.75rem;
      text-align: left;
      font-size: 0.95rem;
    }
    th {
      background: rgba(142, 246, 255, 0.08);
      color: #8ef6ff;
      letter-spacing: 0.02em;
    }
    tr:nth-child(even) {
      background: rgba(142, 246, 255, 0.04);
    }
    .card {
      background: rgba(14, 25, 37, 0.85);
      padding: 1.5rem;
      border-radius: 16px;
      box-shadow: 0 12px 30px rgba(10, 35, 66, 0.35);
      backdrop-filter: blur(8px);
      margin-bottom: 2rem;
    }
    .card h2 {
      margin: 0 0 1rem;
      color: #8ef6ff;
      font-size: 1.35rem;
    }
    label {
      display: block;
      margin-top: 0.75rem;
      font-size: 0.9rem;
      color: #a7c6d9;
    }
    input {
      width: 100%;
      padding: 0.55rem 0.6rem;
      margin-top: 0.35rem;
      border-radius: 8px;
      border: 1px solid rgba(142, 246, 255, 0.3);
      background: rgba(9, 15, 24, 0.9);
      color: #e8f6ff;
      font-size: 0.95rem;
    }
    button {
      margin-top: 1rem;
      padding: 0.65rem 1.4rem;
      border: none;
      border-radius: 999px;
      background: linear-gradient(135deg, #8ef6ff, #7b61ff);
      color: #041121;
      font-weight: 600;
      cursor: pointer;
      box-shadow: 0 6px 18px rgba(123, 97, 255, 0.35);
      transition: transform 0.2s ease, box-shadow 0.2s ease;
    }
    button:hover {
      transform: translateY(-2px);
      box-shadow: 0 10px 22px rgba(123, 97, 255, 0.45);
    }
    .status {
      margin-bottom: 1.5rem;
      padding: 0.9rem 1.2rem;
      border-radius: 14px;
      background: rgba(46, 204, 113, 0.15);
      color: #2ecc71;
      border: 1px solid rgba(46, 204, 113, 0.25);
    }
    .section-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      gap: 1rem;
    }
    .section-header h2 {
      margin: 0;
    }
    .toggle-btn {
      margin-top: 0;
      padding: 0.45rem 1.1rem;
      font-size: 0.85rem;
      background: rgba(142, 246, 255, 0.12);
      color: #8ef6ff;
      border: 1px solid rgba(142, 246, 255, 0.4);
      box-shadow: none;
    }
    .toggle-btn:hover {
      background: rgba(142, 246, 255, 0.2);
      transform: none;
    }
    .section-body.collapsed {
      display: none;
    }
    .overview-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
      gap: 1.1rem;
      margin-top: 1rem;
    }
    .metric-card {
      background: rgba(142, 246, 255, 0.05);
      border-radius: 12px;
      padding: 0.95rem 1.1rem;
      border: 1px solid rgba(142, 246, 255, 0.12);
    }
    .metric-card span {
      display: block;
      color: #a7c6d9;
      font-size: 0.8rem;
      text-transform: uppercase;
      letter-spacing: 0.05em;
      margin-bottom: 0.35rem;
    }
    .metric-card strong {
      font-size: 1.1rem;
      color: #e8f6ff;
    }
    .positions-wrapper {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
      gap: 1.5rem;
    }
    .positions-wrapper table {
      margin-top: 0.5rem;
    }
    .chart-card {
      margin-top: 2rem;
    }
    #profitChart {
      width: 100%;
      min-height: 260px;
    }
    .log-list {
      list-style: none;
      padding: 0;
      margin: 0;
      max-height: 260px;
      overflow-y: auto;
      border-radius: 10px;
      border: 1px solid rgba(142, 246, 255, 0.12);
      background: rgba(8, 20, 34, 0.75);
    }
    .log-list li {
      padding: 0.75rem 1rem;
      border-bottom: 1px solid rgba(142, 246, 255, 0.08);
      font-size: 0.9rem;
    }
    .log-list li:last-child {
      border-bottom: none;
    }
    .log-time {
      color: #8ef6ff;
      font-size: 0.75rem;
      display: block;
      margin-bottom: 0.2rem;
      opacity: 0.8;
    }
    .empty-state {
      padding: 1rem;
      text-align: center;
      color: rgba(232, 246, 255, 0.65);
      border: 1px dashed rgba(142, 246, 255, 0.2);
      border-radius: 12px;
      margin-top: 1rem;
    }
    @media (max-width: 768px) {
      body {
        padding: 1.5rem;
      }
      .card {
        padding: 1.1rem;
      }
    }
  </style>
  <script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.4/dist/chart.umd.min.js" integrity="sha384-nazCNA/GWGbS6mJ//+Q1n1fN7m9W2KeOlw1zcOn3Eh1xUCkoJ6TP2PzefDs2fzGE" crossorigin="anonymous"></script>
</head>
<body>
  <h1>Hyper Trade Strategy Monitor</h1>
  {{if .Updated}}
  <div class="status">Parameters updated successfully.</div>
  {{end}}
  <div class="card">
    <div class="section-header">
      <h2>Strategy Parameters</h2>
      <button type="button" class="toggle-btn" data-toggle="params-section">Collapse</button>
    </div>
    <div class="section-body" id="params-section">
      <form method="POST" action="/params" class="params-form">
        <label>Amplitude Threshold
          <input type="text" name="amplitude" value="{{.Params.AmplitudeThreshold}}" />
        </label>
        <label>Order Amount (USDT)
          <input type="text" name="amount" value="{{.Params.OrderAmount}}" />
        </label>
        <label>Take Profit Ratio
          <input type="text" name="take_profit" value="{{.Params.TakeProfitRatio}}" />
        </label>
        <label>Stop Loss Ratio
          <input type="text" name="stop_loss" value="{{.Params.StopLossRatio}}" />
        </label>
        <label>Leverage
          <input type="text" name="leverage" value="{{.Params.Leverage}}" />
        </label>
        <label>Position TTL (e.g. 5m)
          <input type="text" name="ttl" value="{{.Params.PositionTTL}}" />
        </label>
        <label>Amplitude Window (e.g. 500ms)
          <input type="text" name="window" value="{{.Params.AmplitudeWindow}}" />
        </label>
        <label>Initial Balance (USDT)
          <input type="text" name="initial_balance" value="{{.Params.InitialBalance}}" />
        </label>
        <button type="submit">Update Parameters</button>
      </form>
    </div>
  </div>

  <div class="card">
    <div class="section-header">
      <h2>Strategy Overview</h2>
      <span style="color: rgba(232, 246, 255, 0.65); font-size: 0.9rem;">Uptime: {{formatDuration .Uptime}}</span>
    </div>
    <div class="overview-grid">
      <div class="metric-card">
        <span>Start Time</span>
        <strong>{{formatTime .Snapshot.Summary.StartTime}}</strong>
      </div>
      <div class="metric-card">
        <span>Initial Balance</span>
        <strong>{{formatCurrency .Snapshot.Summary.InitialBalance}} USDT</strong>
      </div>
      <div class="metric-card">
        <span>Current Balance</span>
        <strong>{{formatCurrency .Snapshot.Summary.CurrentBalance}} USDT</strong>
      </div>
      <div class="metric-card">
        <span>Total Profit</span>
        <strong>{{formatCurrency .Snapshot.Summary.TotalProfit}} USDT</strong>
      </div>
      <div class="metric-card">
        <span>Return</span>
        <strong>{{formatPercent (calcReturn .Snapshot.Summary.TotalProfit .Snapshot.Summary.InitialBalance)}}</strong>
      </div>
      <div class="metric-card">
        <span>Trade Volume</span>
        <strong>{{formatCurrency .Snapshot.Summary.TotalVolume}} USDT</strong>
      </div>
      <div class="metric-card">
        <span>Trades Executed</span>
        <strong>{{.Snapshot.Summary.TradeCount}}</strong>
      </div>
    </div>
  </div>

  <div class="card">
    <h2>Symbol Status</h2>
    <table>
      <thead>
        <tr>
          <th>Binance</th>
          <th>Hyperliquid</th>
          <th>Last Price</th>
          <th>Last Amplitude</th>
          <th>Last Update</th>
          <th>Last Order</th>
        </tr>
      </thead>
      <tbody>
        {{range .Snapshot.Symbols}}
        <tr>
          <td>{{.BinanceSymbol}}</td>
          <td>{{.HyperliquidSymbol}}</td>
          <td>{{formatFloat .LastPrice}}</td>
          <td>{{formatPercent .LastAmplitude}}</td>
          <td>{{formatTime .LastUpdate}}</td>
          <td>{{formatTime .LastOrderTime}}</td>
        </tr>
        {{else}}
        <tr><td colspan="6">No market data received yet.</td></tr>
        {{end}}
      </tbody>
    </table>
  </div>

  <div class="card">
    <h2>Positions</h2>
    <div class="positions-wrapper">
      <div>
        <h3>Current Positions</h3>
        {{if .Snapshot.CurrentPositions}}
        <table>
          <thead>
            <tr>
              <th>ID</th>
              <th>Symbol</th>
              <th>Side</th>
              <th>Size</th>
              <th>Price</th>
              <th>Opened</th>
            </tr>
          </thead>
          <tbody>
            {{range .Snapshot.CurrentPositions}}
            <tr>
              <td>{{.ID}}</td>
              <td>{{.Symbol}}</td>
              <td>{{.Side}}</td>
              <td>{{formatFloat .Size}}</td>
              <td>{{formatFloat .EntryPrice}}</td>
              <td>{{formatTime .OpenedAt}}</td>
            </tr>
            {{end}}
          </tbody>
        </table>
        {{else}}
        <div class="empty-state">No open positions at the moment.</div>
        {{end}}
      </div>
      <div>
        <h3>Position History</h3>
        {{if .Snapshot.PositionHistory}}
        <table>
          <thead>
            <tr>
              <th>Symbol</th>
              <th>Side</th>
              <th>Status</th>
              <th>Profit</th>
              <th>Opened</th>
              <th>Closed</th>
            </tr>
          </thead>
          <tbody>
            {{range .Snapshot.PositionHistory}}
            <tr>
              <td>{{.Symbol}}</td>
              <td>{{.Side}}</td>
              <td>{{.Status}}</td>
              <td>{{formatCurrency .Profit}}</td>
              <td>{{formatTime .OpenedAt}}</td>
              <td>{{formatTime .ClosedAt}}</td>
            </tr>
            {{end}}
          </tbody>
        </table>
        {{else}}
        <div class="empty-state">No historical positions recorded yet.</div>
        {{end}}
      </div>
    </div>
    <div class="chart-card">
      <h3>Profit Trend</h3>
      <canvas id="profitChart"></canvas>
    </div>
  </div>

  <div class="card">
    <div class="section-header">
      <h2>Strategy Logs</h2>
      <button type="button" class="toggle-btn" data-toggle="logs-section">Expand</button>
    </div>
    <div class="section-body collapsed" id="logs-section">
      {{if .Snapshot.Logs}}
      <ul class="log-list">
        {{range .Snapshot.Logs}}
        <li>
          <span class="log-time">{{formatTime .Time}}</span>
          {{.Message}}
        </li>
        {{end}}
      </ul>
      {{else}}
      <div class="empty-state">No log messages yet.</div>
      {{end}}
    </div>
  </div>

  <script>
    document.querySelectorAll('[data-toggle]').forEach(function(btn) {
      var targetId = btn.getAttribute('data-toggle');
      var target = document.getElementById(targetId);
      if (!target) {
        return;
      }

      btn.addEventListener('click', function() {
        var isCollapsed = target.classList.toggle('collapsed');
        btn.textContent = isCollapsed ? 'Expand' : 'Collapse';
      });
    });

    var historyData = {{toJSON .Snapshot.PositionHistory}};
    var ctx = document.getElementById('profitChart');
    if (ctx && Array.isArray(historyData)) {
      var labels = [];
      var values = [];
      var cumulative = 0;
      historyData.forEach(function(item) {
        var opened = item.OpenedAt ? new Date(item.OpenedAt) : null;
        labels.push(opened ? opened.toLocaleString() : '');
        cumulative += item.Profit || 0;
        values.push(Number(cumulative.toFixed(4)));
      });

      new Chart(ctx, {
        type: 'line',
        data: {
          labels: labels,
          datasets: [{
            label: 'Cumulative Profit (USDT)',
            data: values,
            fill: true,
            tension: 0.35,
            borderColor: '#8ef6ff',
            backgroundColor: 'rgba(142, 246, 255, 0.15)',
            pointRadius: 2,
          }]
        },
        options: {
          plugins: {
            legend: {
              labels: {
                color: '#e8f6ff'
              }
            }
          },
          scales: {
            x: {
              ticks: { color: '#a7c6d9' },
              grid: { color: 'rgba(142, 246, 255, 0.05)' }
            },
            y: {
              ticks: { color: '#a7c6d9' },
              grid: { color: 'rgba(142, 246, 255, 0.05)' }
            }
          }
        }
      });
    }
  </script>
</body>
</html>
`))

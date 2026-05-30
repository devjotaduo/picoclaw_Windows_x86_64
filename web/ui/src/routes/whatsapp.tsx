// WhatsApp connection page: shows the connection state, a QR code to pair a
// number, and connect/disconnect actions. Incoming messages are answered by
// the default agent. The page polls /api/whatsapp/status while open.

import { useEffect, useRef, useState } from 'react'
import { QRCodeSVG } from 'qrcode.react'
import { whatsappApi, type WhatsAppStatus } from '../api/whatsapp'

export function WhatsAppPage() {
  const [status, setStatus] = useState<WhatsAppStatus | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const timer = useRef<number | null>(null)

  const refresh = async () => {
    try {
      setStatus(await whatsappApi.status())
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  useEffect(() => {
    void refresh()
    timer.current = window.setInterval(refresh, 2500)
    return () => {
      if (timer.current) window.clearInterval(timer.current)
    }
  }, [])

  const connect = async () => {
    setBusy(true)
    setError(null)
    try {
      await whatsappApi.connect()
      await refresh()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  const logout = async () => {
    if (!confirm('Desconectar este WhatsApp? Será preciso ler o QR de novo.')) return
    setBusy(true)
    setError(null)
    try {
      await whatsappApi.logout()
      await refresh()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  const state = status?.state

  return (
    <div className="page whatsapp-page">
      <h1>WhatsApp</h1>
      <p className="muted">
        Conecte um número de WhatsApp lendo o QR. As mensagens recebidas são respondidas pelo agente
        padrão. A sessão fica salva no servidor e reconecta sozinha após reinícios.
      </p>

      {status && !status.available && (
        <p className="error">
          WhatsApp indisponível neste servidor (a sessão precisa de um volume de dados gravável em /data).
        </p>
      )}
      {error && <p className="error">{error}</p>}

      {status?.available && (
        <div className="card whatsapp-card">
          <div className="whatsapp-state">
            Status:{' '}
            <strong className={`wa-${state}`}>
              {state === 'connected'
                ? '🟢 Conectado'
                : state === 'qr'
                  ? '🟡 Aguardando leitura do QR'
                  : '⚪ Desconectado'}
            </strong>
          </div>

          {state === 'qr' && status.qr && (
            <div className="whatsapp-qr">
              <QRCodeSVG value={status.qr} size={240} marginSize={3} />
              <p className="muted">
                No celular: WhatsApp → <b>Aparelhos conectados</b> → <b>Conectar um aparelho</b> → aponte
                a câmera para este QR.
              </p>
            </div>
          )}
          {state === 'qr' && !status.qr && <p className="muted">Gerando QR…</p>}

          <div className="whatsapp-actions">
            {state !== 'connected' && (
              <button onClick={connect} disabled={busy}>
                {busy ? 'Conectando…' : status.registered ? 'Reconectar' : 'Conectar / Gerar QR'}
              </button>
            )}
            {(state === 'connected' || status.registered) && (
              <button className="danger" onClick={logout} disabled={busy}>
                Desconectar
              </button>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

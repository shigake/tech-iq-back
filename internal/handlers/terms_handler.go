package handlers

import (
	"github.com/gofiber/fiber/v2"
)

type TermsHandler struct{}

func NewTermsHandler() *TermsHandler {
	return &TermsHandler{}
}

// TermsContent represents the terms of service content
type TermsContent struct {
	Title         string `json:"title"`
	LastUpdated   string `json:"lastUpdated"`
	Content       string `json:"content"`
	AcceptanceMsg string `json:"acceptanceMsg"`
}

// GetTerms returns the terms of service
func (h *TermsHandler) GetTerms(c *fiber.Ctx) error {
	terms := TermsContent{
		Title:       "Termos de Serviço - TechERP",
		LastUpdated: "Janeiro 2025",
		Content: `
## 1. ACEITAÇÃO DOS TERMOS

Ao acessar e usar o sistema TechERP, você concorda em cumprir estes Termos de Serviço. Se você não concordar com qualquer parte destes termos, não use o sistema.

## 2. DESCRIÇÃO DO SERVIÇO

O TechERP é um sistema de gestão empresarial que oferece:
- Gestão de técnicos e funcionários
- Controle de tickets e chamados
- Gestão de clientes
- Dashboard e relatórios
- Sistema de autenticação e autorização

## 3. RESPONSABILIDADES DO USUÁRIO

### 3.1 Credenciais de Acesso
- Manter a confidencialidade de suas credenciais de login
- Não compartilhar sua conta com terceiros
- Notificar imediatamente sobre uso não autorizado

### 3.2 Uso Apropriado
- Usar o sistema apenas para fins empresariais legítimos
- Não tentar burlar as medidas de segurança
- Respeitar os níveis de acesso definidos pelo administrador

## 4. NÍVEIS DE ACESSO

### 4.1 Administrador (ADMIN)
- Acesso total ao sistema
- Pode criar, editar e excluir todos os dados
- Gerencia usuários e permissões

### 4.2 Funcionário (EMPLOYEE)
- Acesso de leitura e escrita a dados operacionais
- Pode criar, editar tickets, clientes e dados relacionados
- Não pode gerenciar usuários

### 4.3 Usuário/Técnico (USER)
- Acesso somente leitura
- Pode visualizar dados mas não modificar
- Ideal para técnicos em campo

## 5. PROTEÇÃO DE DADOS

### 5.1 Privacidade
- Os dados inseridos no sistema são de propriedade da empresa
- Implementamos medidas de segurança para proteger as informações
- Acesso aos dados é restrito conforme o nível do usuário

### 5.2 Backup e Recuperação
- Realizamos backups regulares dos dados
- Implementamos medidas de recuperação em caso de falhas
- Os dados podem ser exportados em formato Excel/CSV

## 6. LIMITAÇÕES DE RESPONSABILIDADE

### 6.1 Disponibilidade do Serviço
- Fazemos nosso melhor para manter o sistema disponível 24/7
- Não garantimos disponibilidade ininterrupta
- Podem ocorrer manutenções programadas

### 6.2 Integridade dos Dados
- O usuário é responsável pela veracidade dos dados inseridos
- Recomendamos backups regulares dos dados importantes
- Não nos responsabilizamos por perda de dados por uso inadequado

## 7. MODIFICAÇÕES DOS TERMOS

Reservamos o direito de modificar estes termos a qualquer momento. As alterações entrarão em vigor imediatamente após a publicação. O uso continuado do sistema após as alterações constitui aceitação dos novos termos.

## 8. SUPORTE E CONTATO

Para suporte técnico ou dúvidas sobre estes termos:
- Email: suporte@techerp.com
- Sistema: Use o menu de ajuda no painel administrativo

## 9. LEI APLICÁVEL

Estes termos são regidos pela legislação brasileira. Qualquer disputa será resolvida no foro da comarca onde está localizada a sede da empresa.

---

**Data de vigência:** Janeiro de 2025
**Versão:** 1.0
`,
		AcceptanceMsg: "Ao continuar usando o sistema, você concorda com estes Termos de Serviço.",
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    terms,
	})
}

// AcceptTerms records user acceptance of terms (for future implementation)
func (h *TermsHandler) AcceptTerms(c *fiber.Ctx) error {
	// This could be implemented to track user acceptance in the database
	// For now, we just return success
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Termos aceitos com sucesso",
	})
}
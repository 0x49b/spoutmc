import React, { useEffect, useMemo, useState } from 'react';
import axios from 'axios';
import {
  Alert,
  Checkbox,
  ClipboardCopy,
  Form,
  FormGroup,
  FormSelect,
  FormSelectOption,
  PageSection,
  TextInput,
  Title,
  Wizard,
  WizardHeader,
  WizardStep
} from '@patternfly/react-core';
import { completeSetup, getSetupStatus } from '../../service/apiService';

const EULA_LANGUAGES = [
  { value: 'en-en', label: 'English' },
  { value: 'de-de', label: 'Deutsch' },
  { value: 'fr-fr', label: 'Français' },
  { value: 'es-es', label: 'Español' },
  { value: 'it-it', label: 'Italiano' },
  { value: 'pt-br', label: 'Português (Brasil)' },
  { value: 'ja-jp', label: '日本語' },
  { value: 'ko-kr', label: '한국어' }
];

const EULA_LANGUAGE_STORAGE_KEY = 'setup_wizard_eula_language';
const DEFAULT_EULA_LANGUAGE = 'en-en';

const normalizeLanguageSelection = (language?: string | null): string => {
  const normalized = (language || '').toLowerCase();
  const exists = EULA_LANGUAGES.some((option) => option.value === normalized);
  return exists ? normalized : DEFAULT_EULA_LANGUAGE;
};

const getInitialEulaLanguage = (): string => {
  const stored = localStorage.getItem(EULA_LANGUAGE_STORAGE_KEY);
  if (stored) {
    return normalizeLanguageSelection(stored);
  }
  const browserLanguage = typeof navigator !== 'undefined' ? navigator.language : DEFAULT_EULA_LANGUAGE;
  return normalizeLanguageSelection(browserLanguage);
};

const resolveEulaAssetUrl = (language: string): string =>
  new URL(`../../assets/eula/${language}.html`, import.meta.url).toString();

const SetupWizard: React.FC = () => {
  const [eulaLanguage, setEulaLanguage] = useState(getInitialEulaLanguage);
  const [eulaHtml, setEulaHtml] = useState('');
  const [eulaLoading, setEulaLoading] = useState(true);
  const [eulaFetchError, setEulaFetchError] = useState('');
  const [eulaTypedValue, setEulaTypedValue] = useState('');
  const [adminAlreadyExists, setAdminAlreadyExists] = useState(false);

  const [adminEmail, setAdminEmail] = useState('');
  const [adminPassword, setAdminPassword] = useState('');
  const [adminDisplayName, setAdminDisplayName] = useState('Admin');

  const [dataPath, setDataPath] = useState('./data');

  const [enableGitOps, setEnableGitOps] = useState(false);
  const [gitPollInterval, setGitPollInterval] = useState('30s');
  const [gitRepository, setGitRepository] = useState('');
  const [gitBranch, setGitBranch] = useState('main');

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    let cancelled = false;
    const loadSetupStatus = async () => {
      try {
        const response = await getSetupStatus();
        if (!cancelled) {
          setAdminAlreadyExists(Boolean(response.data.adminExists));
        }
      } catch {
        // Keep default (admin step visible) on transient status errors.
      }
    };
    loadSetupStatus();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    localStorage.setItem(EULA_LANGUAGE_STORAGE_KEY, eulaLanguage);
  }, [eulaLanguage]);
  
  useEffect(() => {
    let cancelled = false;
    const loadEula = async () => {
      setEulaLoading(true);
      setEulaFetchError('');
      try {
        const response = await fetch(resolveEulaAssetUrl(eulaLanguage));
        if (!response.ok) {
          throw new Error(`Failed to load local EULA asset (${response.status})`);
        }
        const content = await response.text();
        if (!cancelled) {
          setEulaHtml(content);
        }
      } catch (err: unknown) {
        let message = 'Failed to load local EULA HTML.';
        if (err instanceof Error && err.message) {
          message = err.message;
        }
        if (!cancelled) {
          setEulaFetchError(message);
        }
      } finally {
        if (!cancelled) {
          setEulaLoading(false);
        }
      }
    };

    loadEula();
    return () => {
      cancelled = true;
    };
  }, [eulaLanguage]);

  const eulaAccepted = eulaTypedValue.trim().toLowerCase() === 'accept';
  const eulaStepValid = !eulaLoading && !eulaFetchError && eulaHtml.trim() !== '' && eulaAccepted;
  const adminStepValid = adminAlreadyExists || (adminEmail.trim() !== '' && adminPassword.trim().length >= 6);
  const storageStepValid = dataPath.trim() !== '';
  const gitStepValid = !enableGitOps || (
    gitPollInterval.trim() !== '' &&
    gitRepository.trim() !== '' &&
    gitBranch.trim() !== ''
  );
  const canSubmit = eulaStepValid && adminStepValid && storageStepValid && gitStepValid;

  const submitErrorFromUnknown = (err: unknown): string => {
    if (axios.isAxiosError(err)) {
      return err.response?.data?.error || err.message || 'Setup failed';
    }
    if (err instanceof Error) {
      return err.message;
    }
    return 'Setup failed';
  };

  const handleComplete = async (_event: React.MouseEvent<HTMLButtonElement>) => {
    setError('');
    if (!canSubmit) {
      setError('Please complete all required wizard steps before finishing setup.');
      return;
    }
    setLoading(true);
    try {
      await completeSetup({
        dataPath: dataPath.trim(),
        acceptEula: true,
        adminEmail: adminAlreadyExists ? undefined : adminEmail.trim(),
        adminPassword: adminAlreadyExists ? undefined : adminPassword.trim(),
        adminDisplayName: adminAlreadyExists ? undefined : adminDisplayName.trim() || undefined,
        enableGitOps,
        gitPollInterval: enableGitOps ? gitPollInterval.trim() : undefined,
        gitRepository: enableGitOps ? gitRepository.trim() : undefined,
        gitBranch: enableGitOps ? gitBranch.trim() : undefined
      });
      // Force app re-initialization so setup gate is re-evaluated before showing login.
      window.location.assign('/login');
    } catch (e: unknown) {
      setError(submitErrorFromUnknown(e));
    } finally {
      setLoading(false);
    }
  };

  const eulaStepContent = useMemo(() => {
    if (eulaLoading) {
      return <p>Loading Minecraft EULA text...</p>;
    }
    if (eulaFetchError) {
      return <Alert variant="danger" title={eulaFetchError} />;
    }
    return (
      <Form>
        <FormGroup label="EULA language" fieldId="minecraft-eula-language">
          <FormSelect
            id="minecraft-eula-language"
            value={eulaLanguage}
            onChange={(_event, value) => setEulaLanguage(String(value))}
            aria-label="Minecraft EULA language"
          >
            {EULA_LANGUAGES.map((option) => (
              <FormSelectOption key={option.value} value={option.value} label={option.label} />
            ))}
          </FormSelect>
        </FormGroup>
        <FormGroup label="Minecraft EULA text" fieldId="minecraft-eula-text">
          <div
            id="minecraft-eula-text"
            style={{
              minHeight: '18rem',
              maxHeight: '22rem',
              overflow: 'auto',
              border: '1px solid var(--pf-v6-global--BorderColor--100)',
              borderRadius: 'var(--pf-v6-global--BorderRadius--sm)',
              padding: 'var(--pf-v6-global--spacer--md)',
              backgroundColor: 'var(--pf-v6-global--BackgroundColor--100)'
            }}
            dangerouslySetInnerHTML={{ __html: eulaHtml }}
          />
        </FormGroup>
        <FormGroup label="Type accept to continue" fieldId="minecraft-eula-accept">
          <TextInput
            id="minecraft-eula-accept"
            value={eulaTypedValue}
            onChange={(_event, value) => setEulaTypedValue(value)}
            isRequired
          />
          <p className="pf-v6-u-mt-xs">
            Type exactly accept to confirm that you accept the Minecraft EULA.
          </p>
        </FormGroup>
      </Form>
    );
  }, [eulaFetchError, eulaHtml, eulaLoading, eulaTypedValue, eulaLanguage]);

  return (
    <PageSection>
      <div style={{ maxWidth: '960px', margin: '0 auto' }}>
        <Title headingLevel="h1" size="2xl" className="pf-v6-u-mb-md">
          Welcome to SpoutMC
        </Title>
        <p className="pf-v6-u-mb-md">Complete this setup wizard to finish first-time configuration.</p>
        {error && <Alert variant="danger" title={error} className="pf-v6-u-mb-md" />}
        <Wizard
          navAriaLabel="Setup wizard steps"
          height={640}
          isVisitRequired
          isProgressive
          onSave={handleComplete}
          onClose={() => undefined}
          header={
            <WizardHeader
              title="SpoutMC setup"
              description="Configure EULA, admin account, storage, and optional GitOps."
              isCloseHidden
            />
          }
        >
          <WizardStep
            id="eula"
            name="Accept Minecraft EULA"
            footer={{
              isNextDisabled: loading || !eulaStepValid,
              nextButtonText: 'Next'
            }}
          >
            {eulaStepContent}
          </WizardStep>
          <WizardStep
            id="admin"
            name="Administrator account"
            isHidden={adminAlreadyExists}
            isDisabled={!eulaStepValid}
            footer={{
              isNextDisabled: loading || !adminStepValid,
              nextButtonText: 'Next'
            }}
          >
            <Form>
              <FormGroup label="Admin email" fieldId="admin-email" isRequired>
                <TextInput
                  id="admin-email"
                  type="email"
                  value={adminEmail}
                  onChange={(_event, value) => setAdminEmail(value)}
                  isRequired
                />
              </FormGroup>
              <FormGroup
                label="Admin password"
                fieldId="admin-password"
                isRequired
              >
                <TextInput
                  id="admin-password"
                  type="password"
                  value={adminPassword}
                  onChange={(_event, value) => setAdminPassword(value)}
                  isRequired
                />
                <p className="pf-v6-u-mt-xs">Minimum 6 characters.</p>
              </FormGroup>
              <FormGroup label="Display name" fieldId="admin-displayname">
                <TextInput
                  id="admin-displayname"
                  value={adminDisplayName}
                  onChange={(_event, value) => setAdminDisplayName(value)}
                />
              </FormGroup>
            </Form>
          </WizardStep>
          <WizardStep
            id="storage"
            name="Storage path"
            isDisabled={!eulaStepValid || !adminStepValid}
            footer={{
              isNextDisabled: loading || !storageStepValid,
              nextButtonText: 'Next'
            }}
          >
            <Form>
              <FormGroup
                label="storage.data_path"
                fieldId="data-path"
                isRequired
              >
                <TextInput
                  id="data-path"
                  value={dataPath}
                  onChange={(_event, value) => setDataPath(value)}
                  isRequired
                />
                <p className="pf-v6-u-mt-xs">Directory where server data is stored.</p>
              </FormGroup>
            </Form>
          </WizardStep>
          <WizardStep
            id="gitops"
            name="GitOps (optional)"
            isDisabled={!eulaStepValid || !adminStepValid || !storageStepValid}
            footer={{
              isNextDisabled: loading || !gitStepValid || !canSubmit,
              nextButtonText: loading ? 'Completing...' : 'Complete setup'
            }}
          >
            <Form>
              <FormGroup fieldId="enable-gitops">
                <Checkbox
                  id="enable-gitops"
                  label="Enable GitOps"
                  isChecked={enableGitOps}
                  onChange={(_event, checked) => setEnableGitOps(checked)}
                />
              </FormGroup>
              {enableGitOps && (
                <>
                  <FormGroup
                    label="git.poll_interval"
                    fieldId="git-poll-interval"
                    isRequired
                  >
                    <TextInput
                      id="git-poll-interval"
                      value={gitPollInterval}
                      onChange={(_event, value) => setGitPollInterval(value)}
                      isRequired
                    />
                    <p className="pf-v6-u-mt-xs">Duration format, for example: 30s, 1m, 5m.</p>
                  </FormGroup>
                  <FormGroup label="git.repository" fieldId="git-repository" isRequired>
                    <TextInput
                      id="git-repository"
                      value={gitRepository}
                      onChange={(_event, value) => setGitRepository(value)}
                      isRequired
                    />
                  </FormGroup>
                  <FormGroup label="git.branch" fieldId="git-branch" isRequired>
                    <TextInput
                      id="git-branch"
                      value={gitBranch}
                      onChange={(_event, value) => setGitBranch(value)}
                      isRequired
                    />
                  </FormGroup>
                  <Alert
                    variant="info"
                    isInline
                    title="Git credentials and webhook secret are configured via environment variables."
                  >
                    <p>
                      Set the following variables for GitOps authentication and webhook validation:
                    </p>
                    <ClipboardCopy isReadOnly hoverTip="Copy" clickTip="Copied">
                      SPOUTMC_GIT_TOKEN
                    </ClipboardCopy>
                    <ClipboardCopy isReadOnly hoverTip="Copy" clickTip="Copied">
                      SPOUTMC_WEBHOOK_SECRET
                    </ClipboardCopy>
                  </Alert>
                </>
              )}
            </Form>
          </WizardStep>
        </Wizard>
      </div>
    </PageSection>
  );
};

export default SetupWizard;
